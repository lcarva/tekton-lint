package validator

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/hashicorp/go-multierror"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/resolution/v1beta1"
	"github.com/tektoncd/pipeline/pkg/remoteresolution/resolver/git"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/bundle"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	knativeclient "knative.dev/pkg/client/injection/kube/client"
	"sigs.k8s.io/yaml"
)

// resolveParameterExpressions resolves Tekton parameter expressions like $(params.paramName) with runtime values
func resolveParameterExpressions(input string, runtimeParams map[string]string) string {
	// Match $(params.paramName) pattern
	re := regexp.MustCompile(`\$\(params\.([^)]+)\)`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract parameter name from the match
		paramName := re.FindStringSubmatch(match)[1]
		if value, exists := runtimeParams[paramName]; exists {
			return value
		}
		// If no runtime value provided, return the original expression
		return match
	})
}

// resolveParamsInTaskRefParams resolves parameter expressions in TaskRef.Params
func resolveParamsInTaskRefParams(params v1.Params, runtimeParams map[string]string) v1.Params {
	if len(runtimeParams) == 0 {
		return params
	}

	resolvedParams := make(v1.Params, len(params))
	for i, param := range params {
		resolvedParam := param.DeepCopy()
		if resolvedParam.Value.StringVal != "" {
			resolvedParam.Value.StringVal = resolveParameterExpressions(resolvedParam.Value.StringVal, runtimeParams)
		}
		resolvedParams[i] = *resolvedParam
	}
	return resolvedParams
}

func ValidatePipeline(ctx context.Context, p v1.Pipeline, runtimeParams map[string]string) error {

	if err := p.Validate(ctx); err != nil {
		var allErrors error
		for _, e := range err.WrappedErrors() {
			details := e.Details
			if len(details) > 0 {
				details = " " + details
			}
			message := strings.TrimSuffix(e.Message, ": ")
			for _, p := range e.Paths {
				allErrors = multierror.Append(allErrors, fmt.Errorf("%v: %v%v", message, p, details))
			}
			if len(e.Paths) == 0 {
				allErrors = multierror.Append(allErrors, fmt.Errorf("%v: %v", message, details))
			}
		}
		return allErrors
	}

	allTaskResults := map[string][]v1.TaskResult{}
	allTaskResultRefs := map[string][]*v1.ResultRef{}

	pipelineTasks := make([]v1.PipelineTask, 0, len(p.Spec.Tasks)+len(p.Spec.Finally))
	pipelineTasks = append(pipelineTasks, p.Spec.Tasks...)
	pipelineTasks = append(pipelineTasks, p.Spec.Finally...)

	for i, pipelineTask := range pipelineTasks {
		fmt.Printf("%d: %s\n", i, pipelineTask.Name)
		allTaskResultRefs[pipelineTask.Name] = v1.PipelineTaskResultRefs(&pipelineTask)
		params := pipelineTask.Params

		taskSpec, err := taskSpecFromPipelineTask(ctx, pipelineTask, runtimeParams)
		if err != nil {
			return fmt.Errorf("retrieving task spec from %s pipeline task: %w", pipelineTask.Name, err)
		}

		paramSpecs := taskSpec.Params
		allTaskResults[pipelineTask.Name] = taskSpec.Results

		// Matrix parameters are not present in pipelineTask.Params at authoring time.
		// Tekton expands matrix values into concrete TaskRuns at runtime, providing
		// the matrix parameters to each expanded TaskRun. For validation purposes,
		// synthesize parameter entries for any parameters provided via matrix so
		// that required-parameter checks pass and type checks align with the Task spec.
		effectiveParams := make(v1.Params, 0, len(params))
		effectiveParams = append(effectiveParams, params...)
		if pipelineTask.Matrix != nil {
			for _, matrixParam := range pipelineTask.Matrix.Params {
				// Skip if already provided explicitly
				if _, found := getPipelineTaskParam(matrixParam.Name, effectiveParams); found {
					continue
				}
				// Find the expected type from the Task spec and synthesize a placeholder value
				if specParam, found := getTaskParam(matrixParam.Name, paramSpecs); found {
					pv := v1.ParamValue{Type: specParam.Type}
					effectiveParams = append(effectiveParams, v1.Param{Name: matrixParam.Name, Value: pv})
				}
			}
		}

		if err := ValidateParameters(effectiveParams, paramSpecs); err != nil {
			return fmt.Errorf("ERROR: %s PipelineTask: %s", pipelineTask.Name, err)
		}
	}

	// Verify result references in PipelineTasks are valid.
	for pipelineTaskName, resultRefs := range allTaskResultRefs {
		if err := ValidateResults(resultRefs, allTaskResults); err != nil {
			return fmt.Errorf("%s PipelineTask results: %w", pipelineTaskName, err)
		}
	}

	// Verify result references in Pipeline are valid.
	for _, pipelineResult := range p.Spec.Results {
		expressions, _ := pipelineResult.GetVarSubstitutionExpressions()
		resultRefs := v1.NewResultRefs(expressions)
		if err := ValidateResults(resultRefs, allTaskResults); err != nil {
			return fmt.Errorf("pipeline results: %w", err)
		}
	}

	return nil
}

func taskSpecFromPipelineTask(ctx context.Context, pipelineTask v1.PipelineTask, runtimeParams map[string]string) (*v1.TaskSpec, error) {
	// Embedded task spec
	if pipelineTask.TaskSpec != nil {
		// Custom Tasks are not supported
		if pipelineTask.TaskSpec.IsCustomTask() {
			return nil, errors.New("custom Tasks are not supported")
		}
		return &pipelineTask.TaskSpec.TaskSpec, nil
	}

	resolvedParams := resolveParamsInTaskRefParams(pipelineTask.TaskRef.Params, runtimeParams)

	var err error
	// A kube client is needed for the resolvers even when no kubernetes interaction is made.
	ctx, err = injectDummyKubeClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("injecting kube client: %w", err)
	}

	if pipelineTask.TaskRef != nil && pipelineTask.TaskRef.Resolver == "bundles" {
		opts, err := bundleResolverOptions(ctx, resolvedParams)
		if err != nil {
			return nil, err
		}

		resolvedResource, err := bundle.GetEntry(ctx, authn.DefaultKeychain, opts)
		if err != nil {
			return nil, err
		}

		var t v1.Task
		if err := yaml.Unmarshal(resolvedResource.Data(), &t); err != nil {
			return nil, err
		}

		return &t.Spec, nil
	}

	if pipelineTask.TaskRef != nil && pipelineTask.TaskRef.Resolver == "git" {
		resolver := git.Resolver{}
		if err := resolver.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("initializing git resolver: %w", err)
		}

		resolvedResource, err := resolver.Resolve(ctx, &v1beta1.ResolutionRequestSpec{Params: resolvedParams})
		if err != nil {
			return nil, fmt.Errorf("resolving git: %w", err)
		}

		var t v1.Task
		if err := yaml.Unmarshal(resolvedResource.Data(), &t); err != nil {
			return nil, err
		}

		return &t.Spec, nil
	}

	return nil, errors.New("unable to retrieve spec for pipeline task")
}

func bundleResolverOptions(ctx context.Context, params v1.Params) (bundle.RequestOptions, error) {
	var allParams v1.Params

	// The "serviceAccount" param is required by the resolver, but it's rarely ever set on Pipeline
	// definitions. Add a default value if one is not set.
	hasSAParam := false
	for _, p := range params {
		if p.Name == bundle.ParamServiceAccount {
			hasSAParam = true
			break
		}
	}
	if !hasSAParam {
		allParams = append(allParams, v1.Param{
			Name: bundle.ParamServiceAccount, Value: *v1.NewStructuredValues("none"),
		})
	}

	allParams = append(allParams, params...)
	return bundle.OptionsFromParams(ctx, allParams)
}

// injectDummyKubeClient creates and adds a kube client to the context.
func injectDummyKubeClient(ctx context.Context) (context.Context, error) {
	config, err := clientcmd.BuildConfigFromFlags("IGNORED", "")
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}

	kubectl, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating new for config: %w", err)
	}

	return context.WithValue(ctx, knativeclient.Key{}, kubectl), nil
}
