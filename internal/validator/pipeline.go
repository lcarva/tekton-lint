package validator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/hashicorp/go-multierror"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/bundle"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/git"
	"sigs.k8s.io/yaml"
)

func ValidatePipeline(ctx context.Context, p v1.Pipeline) error {

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

		taskSpec, err := TaskSpecFromPipelineTask(ctx, pipelineTask)
		if err != nil {
			return fmt.Errorf("retrieving task spec from %s pipeline task: %w", pipelineTask.Name, err)
		}

		paramSpecs := taskSpec.Params
		allTaskResults[pipelineTask.Name] = taskSpec.Results

		if err := ValidateParameters(params, paramSpecs); err != nil {
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

func TaskSpecFromPipelineTask(ctx context.Context, pipelineTask v1.PipelineTask) (*v1.TaskSpec, error) {
	// Embedded task spec
	if pipelineTask.TaskSpec != nil {
		// Custom Tasks are not supported
		if pipelineTask.TaskSpec.IsCustomTask() {
			return nil, errors.New("custom Tasks are not supported")
		}
		return &pipelineTask.TaskSpec.TaskSpec, nil
	}

	if pipelineTask.TaskRef != nil && pipelineTask.TaskRef.Resolver == "bundles" {
		opts, err := bundleResolverOptions(ctx, pipelineTask.TaskRef.Params)
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
		params, err := git.PopulateDefaultParams(ctx, pipelineTask.TaskRef.Params)
		if err != nil {
			return nil, err
		}

		resolvedResource, err := git.ResolveAnonymousGit(ctx, params)
		if err != nil {
			return nil, err
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
