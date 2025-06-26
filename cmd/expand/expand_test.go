package expand

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestExpandCmd_ValidPipeline(t *testing.T) {
	// Create a temporary file with a valid pipeline
	tempDir := t.TempDir()
	pipelineFile := filepath.Join(tempDir, "pipeline.yaml")
	
	pipelineYAML := `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  tasks:
  - name: inline-task
    taskSpec:
      steps:
      - name: echo
        image: alpine
        script: echo 'hello world'
`

	err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a new command instance for testing
	cmd := &cobra.Command{}
	cmd.SetArgs([]string{pipelineFile})
	
	// Capture stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = originalStdout }()

	// Run the command
	err = run(context.Background(), pipelineFile)
	w.Close()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Read the output
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	// Verify the output contains expected YAML
	expectedContent := []string{
		"apiVersion: tekton.dev/v1",
		"kind: Pipeline",
		"metadata:",
		"name: test-pipeline",
		"spec:",
		"tasks:",
		"name: inline-task",
		"taskSpec:",
		"steps:",
		"name: echo",
		"image: alpine",
		"script: echo 'hello world'",
	}

	for _, expected := range expectedContent {
		if !contains(outputStr, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't", expected)
		}
	}
}

func TestExpandCmd_ValidPipelineRun(t *testing.T) {
	// Create a temporary file with a valid PipelineRun
	tempDir := t.TempDir()
	pipelineRunFile := filepath.Join(tempDir, "pipelinerun.yaml")
	
	pipelineRunYAML := `apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: test-pipelinerun
spec:
  pipelineSpec:
    tasks:
    - name: inline-task
      taskSpec:
        steps:
        - name: echo
          image: alpine
          script: echo 'hello world'
`

	err := os.WriteFile(pipelineRunFile, []byte(pipelineRunYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Capture stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = originalStdout }()

	// Run the command
	err = run(context.Background(), pipelineRunFile)
	w.Close()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Read the output
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	// Verify the output contains expected YAML
	expectedContent := []string{
		"apiVersion: tekton.dev/v1",
		"kind: PipelineRun",
		"metadata:",
		"name: test-pipelinerun",
		"spec:",
		"pipelineSpec:",
		"tasks:",
		"name: inline-task",
		"taskSpec:",
		"steps:",
		"name: echo",
		"image: alpine",
		"script: echo 'hello world'",
	}

	for _, expected := range expectedContent {
		if !contains(outputStr, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't", expected)
		}
	}
}

func TestExpandCmd_UnsupportedResource(t *testing.T) {
	// Create a temporary file with an unsupported resource
	tempDir := t.TempDir()
	taskFile := filepath.Join(tempDir, "task.yaml")
	
	taskYAML := `apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: test-task
spec:
  steps:
  - name: echo
    image: alpine
    script: echo 'hello world'
`

	err := os.WriteFile(taskFile, []byte(taskYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run the command
	err = run(context.Background(), taskFile)

	if err == nil {
		t.Fatal("Expected error for unsupported resource type, got none")
	}

	expectedErrorMsg := "tekton.dev/v1/Task is not supported for expansion"
	if err.Error() != expectedErrorMsg {
		t.Fatalf("Expected error '%s', got: %v", expectedErrorMsg, err)
	}
}

func TestExpandCmd_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.yaml")
	
	invalidYAML := `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  tasks:
  - name: inline-task
    taskSpec:
      steps:
      - name: echo
        image: alpine
        script: echo 'hello world'
invalid: yaml: here
`

	err := os.WriteFile(invalidFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run the command
	err = run(context.Background(), invalidFile)

	if err == nil {
		t.Fatal("Expected error for invalid YAML, got none")
	}

	expectedErrorMsg := "unmarshalling"
	if !contains(err.Error(), expectedErrorMsg) {
		t.Fatalf("Expected error to contain '%s', got: %v", expectedErrorMsg, err)
	}
}

func TestExpandCmd_NonExistentFile(t *testing.T) {
	// Test with a non-existent file
	nonExistentFile := "/tmp/non-existent-file.yaml"

	// Run the command
	err := run(context.Background(), nonExistentFile)

	if err == nil {
		t.Fatal("Expected error for non-existent file, got none")
	}

	expectedErrorMsg := "reading"
	if !contains(err.Error(), expectedErrorMsg) {
		t.Fatalf("Expected error to contain '%s', got: %v", expectedErrorMsg, err)
	}
}

func TestExpandCmd_ResolverError(t *testing.T) {
	// Create a temporary file with a pipeline that has an invalid resolver
	tempDir := t.TempDir()
	pipelineFile := filepath.Join(tempDir, "pipeline.yaml")
	
	pipelineYAML := `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  tasks:
  - name: resolver-task
    taskRef:
      resolver: bundles
      params:
      - name: bundle
        value: invalid-bundle-reference
      - name: name
        value: test-task
      - name: kind
        value: task
`

	err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run the command
	err = run(context.Background(), pipelineFile)

	if err == nil {
		t.Fatal("Expected error for invalid resolver, got none")
	}

	expectedErrorMsg := "expanding pipeline"
	if !contains(err.Error(), expectedErrorMsg) {
		t.Fatalf("Expected error to contain '%s', got: %v", expectedErrorMsg, err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
} 