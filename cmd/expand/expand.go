package expand

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/lcarva/tektor/internal/expander"
)

var ExpandCmd = &cobra.Command{
	Use:     "expand",
	Short:   "Expand a Tekton resource by resolving all resolvers",
	Example: "tektor expand pipeline.yaml > expanded-pipeline.yaml",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd.Context(), args[0])
	},
}

func run(ctx context.Context, fname string) error {
	fmt.Fprintf(os.Stderr, "Expanding %s\n", fname)
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
		var p v1.Pipeline
		if err := yaml.Unmarshal(f, &p); err != nil {
			return fmt.Errorf("unmarshalling %s as %s: %w", fname, key, err)
		}
		expanded, err := expander.ExpandPipeline(ctx, p)
		if err != nil {
			return fmt.Errorf("expanding pipeline: %w", err)
		}
		output, err := yaml.Marshal(expanded)
		if err != nil {
			return fmt.Errorf("marshaling expanded pipeline: %w", err)
		}
		fmt.Print(string(output))
	case "tekton.dev/v1/PipelineRun":
		var pr v1.PipelineRun
		if err := yaml.Unmarshal(f, &pr); err != nil {
			return fmt.Errorf("unmarshalling %s as %s: %w", fname, key, err)
		}
		expanded, err := expander.ExpandPipelineRun(ctx, pr)
		if err != nil {
			return fmt.Errorf("expanding pipelinerun: %w", err)
		}
		output, err := yaml.Marshal(expanded)
		if err != nil {
			return fmt.Errorf("marshaling expanded pipelinerun: %w", err)
		}
		fmt.Print(string(output))
	default:
		return fmt.Errorf("%s is not supported for expansion", key)
	}

	return nil
} 