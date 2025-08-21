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
* Resolve remote Tasks via [git resolver](https://tekton.dev/docs/pipelines/git-resolver/).
* Specify runtime parameters that may be used in resolver parameter values
* Resolve missing Tasks from a local directory with `--task-dir` when a `TaskRef` names a Task without a resolver

## CLI usage

Validate a Pipeline and search for referenced Tasks in a local directory tree:

```bash
tektor validate path/to/pipeline.yaml --task-dir path/to/tasks
```

Future work:

* Verify workspace usage.
* Verify PipelineRun parameters match parameters from Pipeline definition.
* Verify results are used according to their defined types.
* Remove printf calls and use proper logging.
* Don't fail on first found error.
