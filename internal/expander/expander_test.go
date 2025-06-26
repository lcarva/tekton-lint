package expander

import (
	"context"
	"testing"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExpandPipeline_NoResolvers(t *testing.T) {
	// Test pipeline with inline taskSpec (no resolvers)
	pipeline := v1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pipeline",
		},
		Spec: v1.PipelineSpec{
			Tasks: []v1.PipelineTask{
				{
					Name: "inline-task",
					TaskSpec: &v1.EmbeddedTask{
						TaskSpec: v1.TaskSpec{
							Steps: []v1.Step{
								{
									Name:  "echo",
									Image: "alpine",
									Script: "echo 'hello world'",
								},
							},
						},
					},
				},
			},
		},
	}

	expanded, err := ExpandPipeline(context.Background(), pipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(expanded.Spec.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(expanded.Spec.Tasks))
	}

	task := expanded.Spec.Tasks[0]
	if task.TaskSpec == nil {
		t.Fatal("Expected taskSpec to be preserved")
	}

	if len(task.TaskSpec.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(task.TaskSpec.Steps))
	}

	if task.TaskSpec.Steps[0].Name != "echo" {
		t.Fatalf("Expected step name 'echo', got '%s'", task.TaskSpec.Steps[0].Name)
	}
}

func TestExpandPipeline_MixedInlineAndResolver(t *testing.T) {
	// Test pipeline with both inline taskSpec and taskRef (resolver)
	pipeline := v1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pipeline",
		},
		Spec: v1.PipelineSpec{
			Tasks: []v1.PipelineTask{
				{
					Name: "inline-task",
					TaskSpec: &v1.EmbeddedTask{
						TaskSpec: v1.TaskSpec{
							Steps: []v1.Step{
								{
									Name:  "echo",
									Image: "alpine",
									Script: "echo 'hello world'",
								},
							},
						},
					},
				},
				{
					Name: "resolver-task",
					TaskRef: &v1.TaskRef{
						ResolverRef: v1.ResolverRef{
							Resolver: "bundles",
							Params: []v1.Param{
								{
									Name:  "bundle",
									Value: *v1.NewStructuredValues("invalid-bundle"),
								},
								{
									Name:  "name",
									Value: *v1.NewStructuredValues("test-task"),
								},
								{
									Name:  "kind",
									Value: *v1.NewStructuredValues("task"),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := ExpandPipeline(context.Background(), pipeline)
	if err == nil {
		t.Fatal("Expected error for invalid bundle resolver, got none")
	}

	// Verify the error message contains expected information
	expectedErrorMsg := "expanding task resolver-task"
	if err.Error()[:len(expectedErrorMsg)] != expectedErrorMsg {
		t.Fatalf("Expected error to start with '%s', got: %v", expectedErrorMsg, err)
	}
}

func TestExpandPipeline_FinallyTasks(t *testing.T) {
	// Test pipeline with finally tasks
	pipeline := v1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pipeline",
		},
		Spec: v1.PipelineSpec{
			Tasks: []v1.PipelineTask{
				{
					Name: "main-task",
					TaskSpec: &v1.EmbeddedTask{
						TaskSpec: v1.TaskSpec{
							Steps: []v1.Step{
								{
									Name:  "main",
									Image: "alpine",
									Script: "echo 'main task'",
								},
							},
						},
					},
				},
			},
			Finally: []v1.PipelineTask{
				{
					Name: "finally-task",
					TaskSpec: &v1.EmbeddedTask{
						TaskSpec: v1.TaskSpec{
							Steps: []v1.Step{
								{
									Name:  "cleanup",
									Image: "alpine",
									Script: "echo 'cleanup'",
								},
							},
						},
					},
				},
			},
		},
	}

	expanded, err := ExpandPipeline(context.Background(), pipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(expanded.Spec.Tasks) != 1 {
		t.Fatalf("Expected 1 main task, got %d", len(expanded.Spec.Tasks))
	}

	if len(expanded.Spec.Finally) != 1 {
		t.Fatalf("Expected 1 finally task, got %d", len(expanded.Spec.Finally))
	}

	finallyTask := expanded.Spec.Finally[0]
	if finallyTask.TaskSpec == nil {
		t.Fatal("Expected finally taskSpec to be preserved")
	}

	if len(finallyTask.TaskSpec.Steps) != 1 {
		t.Fatalf("Expected 1 step in finally task, got %d", len(finallyTask.TaskSpec.Steps))
	}

	if finallyTask.TaskSpec.Steps[0].Name != "cleanup" {
		t.Fatalf("Expected finally step name 'cleanup', got '%s'", finallyTask.TaskSpec.Steps[0].Name)
	}
}

func TestExpandPipelineRun_WithInlinePipelineSpec(t *testing.T) {
	// Test PipelineRun with inline PipelineSpec
	pipelineRun := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pipelinerun",
		},
		Spec: v1.PipelineRunSpec{
			PipelineSpec: &v1.PipelineSpec{
				Tasks: []v1.PipelineTask{
					{
						Name: "inline-task",
						TaskSpec: &v1.EmbeddedTask{
							TaskSpec: v1.TaskSpec{
								Steps: []v1.Step{
									{
										Name:  "echo",
										Image: "alpine",
										Script: "echo 'hello world'",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	expanded, err := ExpandPipelineRun(context.Background(), pipelineRun)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if expanded.Spec.PipelineSpec == nil {
		t.Fatal("Expected PipelineSpec to be preserved")
	}

	if len(expanded.Spec.PipelineSpec.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(expanded.Spec.PipelineSpec.Tasks))
	}

	task := expanded.Spec.PipelineSpec.Tasks[0]
	if task.TaskSpec == nil {
		t.Fatal("Expected taskSpec to be preserved")
	}

	if len(task.TaskSpec.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(task.TaskSpec.Steps))
	}

	if task.TaskSpec.Steps[0].Name != "echo" {
		t.Fatalf("Expected step name 'echo', got '%s'", task.TaskSpec.Steps[0].Name)
	}
}

func TestExpandPipelineRun_WithPipelineRef(t *testing.T) {
	// Test PipelineRun with PipelineRef (should not error, but won't expand the referenced pipeline)
	pipelineRun := v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pipelinerun",
		},
		Spec: v1.PipelineRunSpec{
			PipelineRef: &v1.PipelineRef{
				Name: "referenced-pipeline",
			},
		},
	}

	expanded, err := ExpandPipelineRun(context.Background(), pipelineRun)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if expanded.Spec.PipelineRef == nil {
		t.Fatal("Expected PipelineRef to be preserved")
	}

	if expanded.Spec.PipelineRef.Name != "referenced-pipeline" {
		t.Fatalf("Expected PipelineRef name 'referenced-pipeline', got '%s'", expanded.Spec.PipelineRef.Name)
	}
}

func TestExpandPipeline_EmptyPipeline(t *testing.T) {
	// Test empty pipeline
	pipeline := v1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "empty-pipeline",
		},
		Spec: v1.PipelineSpec{
			Tasks: []v1.PipelineTask{},
		},
	}

	expanded, err := ExpandPipeline(context.Background(), pipeline)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(expanded.Spec.Tasks) != 0 {
		t.Fatalf("Expected 0 tasks, got %d", len(expanded.Spec.Tasks))
	}
}

func TestExpandPipeline_NoTaskRefOrTaskSpec(t *testing.T) {
	// Test pipeline task with neither TaskRef nor TaskSpec
	pipeline := v1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pipeline",
		},
		Spec: v1.PipelineSpec{
			Tasks: []v1.PipelineTask{
				{
					Name: "no-ref-task",
					// No TaskRef or TaskSpec
				},
			},
		},
	}

	_, err := ExpandPipeline(context.Background(), pipeline)
	if err == nil {
		t.Fatal("Expected error for task with no TaskRef or TaskSpec, got none")
	}

	// Verify the error message contains expected information
	expectedErrorMsg := "expanding task no-ref-task"
	if err.Error()[:len(expectedErrorMsg)] != expectedErrorMsg {
		t.Fatalf("Expected error to start with '%s', got: %v", expectedErrorMsg, err)
	}
} 