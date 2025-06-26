package expander

import (
	"context"
	"fmt"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/lcarva/tektor/internal/validator"
)

// ExpandPipeline expands a Pipeline by resolving all taskRef resolvers to inline taskSpecs
func ExpandPipeline(ctx context.Context, p v1.Pipeline) (*v1.Pipeline, error) {
	// Create a deep copy to avoid modifying the original
	expanded := p.DeepCopy()

	// Expand tasks in the main tasks list
	for i := range expanded.Spec.Tasks {
		if err := expandPipelineTask(ctx, &expanded.Spec.Tasks[i]); err != nil {
			return nil, fmt.Errorf("expanding task %s: %w", expanded.Spec.Tasks[i].Name, err)
		}
	}

	// Expand tasks in the finally list
	for i := range expanded.Spec.Finally {
		if err := expandPipelineTask(ctx, &expanded.Spec.Finally[i]); err != nil {
			return nil, fmt.Errorf("expanding finally task %s: %w", expanded.Spec.Finally[i].Name, err)
		}
	}

	return expanded, nil
}

// ExpandPipelineRun expands a PipelineRun by resolving all taskRef resolvers to inline taskSpecs
func ExpandPipelineRun(ctx context.Context, pr v1.PipelineRun) (*v1.PipelineRun, error) {
	// Create a deep copy to avoid modifying the original
	expanded := pr.DeepCopy()

	// If the PipelineRun has an inline PipelineSpec, expand it
	if expanded.Spec.PipelineSpec != nil {
		// Create a temporary Pipeline to expand
		tempPipeline := v1.Pipeline{
			Spec: *expanded.Spec.PipelineSpec,
		}
		
		expandedPipeline, err := ExpandPipeline(ctx, tempPipeline)
		if err != nil {
			return nil, fmt.Errorf("expanding inline pipeline spec: %w", err)
		}
		
		// Replace the PipelineSpec with the expanded one
		expanded.Spec.PipelineSpec = &expandedPipeline.Spec
	}

	// If the PipelineRun references a Pipeline, we need to expand that pipeline first
	if expanded.Spec.PipelineRef != nil {
		// For now, we'll assume the pipeline is already expanded or available
		// In a more complete implementation, we might want to fetch and expand the referenced pipeline
		// This would require additional logic to resolve pipeline references
	}

	return expanded, nil
}

// expandPipelineTask expands a single PipelineTask by resolving its taskRef to an inline taskSpec
func expandPipelineTask(ctx context.Context, pipelineTask *v1.PipelineTask) error {
	// If there's already an inline taskSpec, no expansion needed
	if pipelineTask.TaskSpec != nil {
		return nil
	}

	// If there's no taskRef, nothing to expand
	if pipelineTask.TaskRef == nil {
		return fmt.Errorf("task %s has neither TaskRef nor TaskSpec", pipelineTask.Name)
	}

	// Use the existing resolver logic from the validator package
	taskSpec, err := validator.TaskSpecFromPipelineTask(ctx, *pipelineTask)
	if err != nil {
		return fmt.Errorf("resolving task spec: %w", err)
	}

	// Replace the taskRef with the resolved taskSpec
	pipelineTask.TaskSpec = &v1.EmbeddedTask{
		TaskSpec: *taskSpec,
	}
	pipelineTask.TaskRef = nil

	return nil
} 