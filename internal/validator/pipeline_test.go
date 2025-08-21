package validator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/yaml"
)

// Verifies that matrix-supplied params satisfy required Task params and types
// when the Task is embedded via taskSpec and PipelineTask.Params are omitted.
func TestValidatePipeline_MatrixParamsSatisfyTaskParams(t *testing.T) {
	// Prepare a temp directory and write a minimal Pipeline YAML that uses matrix params
	tmpDir, err := os.MkdirTemp("", "pipeline-matrix-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pipelineYAML := `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: matrix-pipeline
spec:
  tasks:
    - name: run
      taskSpec:
        params:
          - name: img
            type: string
          - name: tags
            type: array
        steps:
          - name: s
            image: "$(params.img)"
            script: |
              echo processing
      matrix:
        params:
          - name: img
            value:
              - alpine:3.18
              - alpine:3.19
          - name: tags
            value:
              - latest
              - edge
`

	pipelineFile := filepath.Join(tmpDir, "pipeline.yaml")
	if err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644); err != nil {
		t.Fatalf("failed to write pipeline file: %v", err)
	}

	var p v1.Pipeline
	b, err := os.ReadFile(pipelineFile)
	if err != nil {
		t.Fatalf("failed to read pipeline file: %v", err)
	}
	if err := yaml.Unmarshal(b, &p); err != nil {
		t.Fatalf("failed to unmarshal pipeline: %v", err)
	}

	if err := ValidatePipeline(context.Background(), p, map[string]string{}); err != nil {
		t.Fatalf("expected validation to succeed, got error: %v", err)
	}
}

// Models a PipelineTask similar to Konflux build where PLATFORM is provided via matrix
// and IMAGE via PipelineTask.Params. Ensures PLATFORM is not treated as missing.
func TestValidatePipeline_MatrixPlatformParamPresent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pipeline-matrix-platform-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pipelineYAML := `apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: build-pipeline
spec:
  params:
    - name: build-platforms
      type: array
      default:
        - linux/amd64
        - linux/arm64
    - name: output-image
      type: string
      default: quay.io/example/app:latest
  tasks:
    - name: build-images
      params:
        - name: IMAGE
          value: $(params.output-image)
      matrix:
        params:
          - name: PLATFORM
            value: $(params.build-platforms)
      taskSpec:
        params:
          - name: PLATFORM
            type: array
          - name: IMAGE
            type: string
        steps:
          - name: build
            image: alpine:3.18
            script: |
              echo building $(params.IMAGE)
`

	pipelineFile := filepath.Join(tmpDir, "pipeline.yaml")
	if err := os.WriteFile(pipelineFile, []byte(pipelineYAML), 0644); err != nil {
		t.Fatalf("failed to write pipeline file: %v", err)
	}

	var p v1.Pipeline
	b, err := os.ReadFile(pipelineFile)
	if err != nil {
		t.Fatalf("failed to read pipeline file: %v", err)
	}
	if err := yaml.Unmarshal(b, &p); err != nil {
		t.Fatalf("failed to unmarshal pipeline: %v", err)
	}

	if err := ValidatePipeline(context.Background(), p, map[string]string{}); err != nil {
		t.Fatalf("expected validation to succeed (PLATFORM provided via matrix), got error: %v", err)
	}
}
