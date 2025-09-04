package validate

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/lcarva/tektor/internal/pac"
	"github.com/lcarva/tektor/internal/validator"
)

// UnsupportedResourceError indicates a file is not a supported Tekton resource
type UnsupportedResourceError struct {
	Filename string
	Message  string
}

func (e UnsupportedResourceError) Error() string {
	return e.Message
}

var (
	runtimeParams []string
	pacParams     []string
	taskDir       string
)

var ValidateCmd = &cobra.Command{
	Use:     "validate",
	Short:   "Validate a Tekton resource",
	Example: "tektor validate /tmp/pipeline.yaml --param taskGitUrl=https://github.com/example/repo --pac-param revision=main",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		params, err := parseRuntimeParams(runtimeParams)
		if err != nil {
			return fmt.Errorf("parsing runtime parameters: %w", err)
		}
		pacParamsMap, err := parseRuntimeParams(pacParams)
		if err != nil {
			return fmt.Errorf("parsing PaC parameters: %w", err)
		}
		ctx := cmd.Context()
		if taskDir != "" {
			ctx = validator.WithTaskDir(ctx, taskDir)
		}
		return run(ctx, args[0], params, pacParamsMap)
	},
}

func init() {
	ValidateCmd.Flags().StringArrayVar(&runtimeParams, "param", []string{}, "Runtime parameters in format key=value (can be specified multiple times)")
	ValidateCmd.Flags().StringArrayVar(&pacParams, "pac-param", []string{}, "PaC template parameters in format key=value (can be specified multiple times)")
	ValidateCmd.Flags().StringVar(&taskDir, "task-dir", "", "Directory to recursively search for missing Tasks referenced by the Pipeline")
}

func parseRuntimeParams(params []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, param := range params {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid parameter format: %s (expected key=value)", param)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}

func run(ctx context.Context, fname string, runtimeParams map[string]string, pacParams map[string]string) error {
	fmt.Printf("Validating %s\n", fname)
	f, err := os.ReadFile(fname)
	if err != nil {
		return fmt.Errorf("reading %s: %w", fname, err)
	}

	var o metav1.PartialObjectMetadata
	if err := yaml.Unmarshal(f, &o); err != nil {
		return fmt.Errorf("unmarshalling %s as k8s resource: %w", fname, err)
	}

	key := fmt.Sprintf("%s/%s", o.APIVersion, o.Kind)
	switch key {
	case "tekton.dev/v1/Pipeline":
		// Resolve the pipeline using PaC to handle parameter substitutions and inlined tasks
		// Use runtimeParams for Tekton parameter substitution and pacParams for PaC template substitution
		resolvedPipelineBytes, err := pac.ResolvePipeline(ctx, fname, o.Name, pacParams)
		if err != nil {
			return fmt.Errorf("resolving pipeline with PAC: %w", err)
		}

		var p v1.Pipeline
		if err := yaml.Unmarshal(resolvedPipelineBytes, &p); err != nil {
			return fmt.Errorf("unmarshalling resolved pipeline as %s: %w", key, err)
		}
		if err := validator.ValidatePipeline(ctx, p, runtimeParams); err != nil {
			return err
		}
	case "tekton.dev/v1/PipelineRun":
		// Use runtimeParams for Tekton parameter substitution and pacParams for PaC template substitution
		f, err = pac.ResolvePipelineRun(ctx, fname, o.Name, pacParams)
		if err != nil {
			return fmt.Errorf("resolving with PAC: %w", err)
		}

		var pr v1.PipelineRun
		if err := yaml.Unmarshal(f, &pr); err != nil {
			return fmt.Errorf("unmarshalling %s as %s: %w", fname, key, err)
		}

		if err := validator.ValidatePipelineRun(ctx, pr); err != nil {
			return err
		}
	case "tekton.dev/v1/Task":
		var t v1.Task
		if err := yaml.Unmarshal(f, &t); err != nil {
			return fmt.Errorf("unmarshaling %s as %s: %w", fname, key, err)
		}
		if err := validator.ValidateTaskV1(ctx, t); err != nil {
			return err
		}
	case "tekton.dev/v1beta1/Task":
		var t v1beta1.Task
		if err := yaml.Unmarshal(f, &t); err != nil {
			return fmt.Errorf("unmarshaling %s as %s: %w", fname, key, err)
		}
		if err := validator.ValidateTaskV1Beta1(ctx, t); err != nil {
			return err
		}
	default:
		return UnsupportedResourceError{
			Filename: fname,
			Message:  fmt.Sprintf("%s is not supported as a Tekton resource", fname),
		}
	}

	return nil
}
