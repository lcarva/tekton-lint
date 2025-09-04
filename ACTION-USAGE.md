# Using Tektor GitHub Action in Other Repositories

This document explains how to use the Tektor GitHub Action in other repositories without publishing it to the GitHub Marketplace.

## Method 1: Direct Repository Reference ⭐ Recommended

Reference the action directly from your workflow:

```yaml
name: Validate Tekton Resources
on:
  pull_request:
    paths:
      - '**/*.yaml'
      - '**/*.yml'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Get changed files
        id: changed-files
        uses: tj-actions/changed-files@v44
        with:
          files: |
            **/*.yaml
            **/*.yml

      - name: Checkout Tektor action
        uses: actions/checkout@v4
        with:
          repository: konflux-ci/tektor
          path: .github/actions/tektor
          ref: main  # Use specific tag for stability

      - name: Validate Tekton resources
        uses: ./.github/actions/tektor
        env:
          CHANGED_FILES: ${{ steps.changed-files.outputs.all_changed_files }}
        with:
          fail-on-error: true
          verbose: true
          task-dir: path/to/local/tasks
          params: |
            taskGitUrl=https://github.com/tektoncd/catalog
            gitRevision=main
          pac-params: |
            revision=main
            branch=development
```

## Method 2: Reusable Workflow ⭐ Best Practice

Use the reusable workflow (requires the workflow to be in a public repo or same organization):

```yaml
name: Validate Tekton Resources
on:
  pull_request:
    paths:
      - '**/*.yaml'
      - '**/*.yml'

jobs:
  validate-tekton:
    uses: konflux-ci/tektor/.github/workflows/validate-tekton-reusable.yml@main
    with:
      fail-on-error: true
      verbose: true
      file-patterns: '**/*.yaml,**/*.yml'
```

## Parameter Support

Tektor supports two types of parameters to customize validation behavior:

### Runtime Parameters (`params`)

Runtime parameters are used for Tekton parameter substitution within your pipeline and task definitions. These help resolve parameter references like `$(params.paramName)` during validation. Use the multi-line format with one parameter per line:

```yaml
- name: Validate Tekton resources
  uses: ./.github/actions/tektor
  env:
    CHANGED_FILES: ${{ steps.changed-files.outputs.all_changed_files }}
  with:
    params: |
      taskGitUrl=https://github.com/tektoncd/catalog
      gitRevision=main
      imageTag=v1.0.0
```

### PaC Template Parameters (`pac-params`)

PaC (Pipelines as Code) template parameters are used for PaC template substitution and preprocessing before validation. Use the multi-line format with one parameter per line:

```yaml
- name: Validate Tekton resources
  uses: ./.github/actions/tektor
  env:
    CHANGED_FILES: ${{ steps.changed-files.outputs.all_changed_files }}
  with:
    pac-params: |
      revision=main
      branch=development
      environment=staging
```

### Using Both Parameter Types

You can use both parameter types together for comprehensive validation:

```yaml
- name: Validate Tekton resources with parameters
  uses: ./.github/actions/tektor
  env:
    CHANGED_FILES: ${{ steps.changed-files.outputs.all_changed_files }}
  with:
    fail-on-error: true
    verbose: true
    params: |
      taskGitUrl=https://github.com/tektoncd/catalog
      gitRevision=main
    pac-params: |
      revision=main
      branch=development

### Local Task Directory (`task-dir`)

Use `task-dir` to tell Tektor where to recursively search for missing Tasks referenced by `taskRef.name` in your Pipeline when no resolver is specified. This is useful for testing or when Tasks are not yet available in a cluster or remote resolver.

```yaml
- name: Validate Tekton resources with local tasks
  uses: ./.github/actions/tektor
  env:
    CHANGED_FILES: ${{ steps.changed-files.outputs.all_changed_files }}
  with:
    task-dir: ./tasks
    verbose: true
```
```

### Dynamic Parameter Values

Parameters can be dynamically set using GitHub Actions expressions:

```yaml
- name: Validate Tekton resources with dynamic params
  uses: ./.github/actions/tektor
  env:
    CHANGED_FILES: ${{ steps.changed-files.outputs.all_changed_files }}
  with:
    params: |
      gitRevision=${{ github.sha }}
      branch=${{ github.ref_name }}
      taskGitUrl=${{ github.event.pull_request.head.repo.html_url }}
    pac-params: |
      revision=${{ github.event.pull_request.head.sha || github.sha }}
```

## Advanced Usage Examples

### With Custom Error Handling

```yaml
- name: Validate Tekton resources
  id: validate
  continue-on-error: true
  uses: ./.github/actions/tektor
  env:
    CHANGED_FILES: ${{ steps.changed-files.outputs.all_changed_files }}
  with:
    fail-on-error: false
    verbose: true
    params: |
      gitRevision=${{ github.sha }}
      repoUrl=${{ github.event.repository.clone_url }}

- name: Comment on PR if validation failed
  if: steps.validate.outputs.validation-errors != '0'
  uses: actions/github-script@v7
  with:
    script: |
      github.rest.issues.createComment({
        issue_number: context.issue.number,
        owner: context.repo.owner,
        repo: context.repo.repo,
        body: `⚠️ Tekton validation found ${steps.validate.outputs.validation-errors} error(s) in ${steps.validate.outputs.validated-files} file(s).`
      })
```
