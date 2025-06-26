# Tektor

Why does this thing exist? Because I'm tired of finding out about problems with my Pipeline *after*
I run it.

It is written in go because that is the language used by the Tekton code base. It makes us not have
to re-invent the wheel to perform certain checks.

It currently supports the following:

* Verify PipelineTasks pass all required parameters to Tasks.
* Verify PipelineTasks pass known parameters to Tasks.
* Verify PipelineTasks pass parameters of expected types to Tasks.
* Verify PipelineTasks use known Task results.
* Resolve remote/local Tasks via
  [PaC resolver](https://docs.openshift.com/pipelines/1.11/pac/using-pac-resolver.html),
  [Bundles resolver](https://tekton.dev/docs/pipelines/bundle-resolver/), and embedded Task
  definitions.
* **Expand Tekton pipelines and pipelineruns with resolvers to fully inlined YAML** (see below)

## Expand Command

Tektor can output a fully expanded Tekton Pipeline or PipelineRun, resolving all supported resolvers (bundles, git) and inlining the full `taskSpec` for each task.

### Usage

```sh
tektor expand pipeline.yaml > expanded-pipeline.yaml
```

- Takes a Pipeline or PipelineRun YAML as input
- Resolves all `taskRef` resolvers (bundles, git) to inline `taskSpec`
- Outputs the fully expanded YAML to stdout
- Handles both `Pipeline` and `PipelineRun` resources
- The output can be applied directly to a cluster without needing resolvers

### Example

**Input:**
```yaml
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: example-pipeline
spec:
  tasks:
  - name: build
    taskRef:
      resolver: bundles
      params:
      - name: bundle
        value: quay.io/tekton-catalog/task-buildah:0.1@sha256:abc123
      - name: name
        value: buildah
      - name: kind
        value: task
```

**Command:**
```sh
tektor expand pipeline.yaml > expanded-pipeline.yaml
```

**Output:**
```yaml
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: example-pipeline
spec:
  tasks:
  - name: build
    taskSpec:
      steps:
      - name: build
        image: quay.io/buildah/stable
        script: |
          # ... full task definition from bundle
```

Future work:

* Resolve remote Tasks via [git resolver](https://tekton.dev/docs/pipelines/git-resolver/).
* Verify workspace usage.
* Verify PipelineRun parameters match parameters from Pipeline definition.
* Verify results are used according to their defined types.
* Remove printf calls and use proper logging.
* Don't fail on first found error.
