package validator

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	syaml "sigs.k8s.io/yaml"
)

// context key for task-dir
type contextKey string

const taskDirContextKey contextKey = "validator-task-dir"

var splitRe = regexp.MustCompile(`(?m)^---\s*$`)

// WithTaskDir stores a directory path in context for resolving local Tasks.
func WithTaskDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, taskDirContextKey, dir)
}

func taskDirFromContext(ctx context.Context) string {
	v := ctx.Value(taskDirContextKey)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// findTaskSpecInDir walks the given directory recursively and returns the v1.TaskSpec
// for a Task with the provided name, if found. Only tekton.dev/v1 Tasks are supported.
func findTaskSpecInDir(rootDir string, taskName string) (*v1.TaskSpec, error) {
	if rootDir == "" {
		return nil, nil
	}
	if _, err := os.Stat(rootDir); err != nil {
		return nil, fmt.Errorf("stat --task-dir: %w", err)
	}

	var foundSpec *v1.TaskSpec

	walkErr := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading task file %q: %w", path, err)
		}

		docs := splitRe.Split(string(b), -1)
		for _, doc := range docs {
			doc = strings.TrimSpace(doc)
			if doc == "" {
				continue
			}
			var meta metav1.PartialObjectMetadata
			if err := syaml.Unmarshal([]byte(doc), &meta); err != nil {
				continue
			}
			if meta.Kind != "Task" || meta.APIVersion != "tekton.dev/v1" {
				continue
			}
			if meta.Name != taskName {
				continue
			}
			var t v1.Task
			if err := syaml.Unmarshal([]byte(doc), &t); err != nil {
				// Warn and continue if we fail to unmarshal a candidate Task document
				fmt.Printf("[WARNING] failed to unmarshal Task %q from %s: %v\n", taskName, path, err)
				continue
			}
			foundSpec = &t.Spec
			return fs.SkipAll
		}
		return nil
	})

	if walkErr != nil && walkErr != fs.SkipAll {
		return nil, fmt.Errorf("walking %q: %w", rootDir, walkErr)
	}
	return foundSpec, nil
}
