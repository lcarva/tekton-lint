#!/bin/bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Initialize counters
VALIDATED_COUNT=0
ERROR_COUNT=0
CHECKED_COUNT=0
SKIPPED_COUNT=0

# Function to log messages
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_skipped() {
    echo -e "${YELLOW}[SKIPPED]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}


# Function to validate a single file
validate_file() {
    local file="$1"

    log_info "Validating: $file"

    # Build parameter arguments array
    local param_args=()
    local param_count=0

    # Debug: Show raw TEKTOR_PARAMS
    if [[ "$VERBOSE" == "true" && -n "$TEKTOR_PARAMS" ]]; then
        log_info "Raw TEKTOR_PARAMS: '$TEKTOR_PARAMS'"
    fi

    # Process TEKTOR_PARAMS (runtime parameters) - newline-separated format only
    if [[ -n "$TEKTOR_PARAMS" ]]; then
        while IFS= read -r param; do
            # Trim whitespace
            param=$(echo "$param" | xargs)
            if [[ -n "$param" ]]; then
                param_args+=("--param" "$param")
                param_count=$((param_count + 1))
                if [[ "$VERBOSE" == "true" ]]; then
                    log_info "Added runtime parameter: --param $param"
                fi
            fi
        done <<< "$TEKTOR_PARAMS"
    fi

    # Process TEKTOR_PAC_PARAMS (PaC template parameters) - newline-separated format only
    if [[ -n "$TEKTOR_PAC_PARAMS" ]]; then
        while IFS= read -r param; do
            # Trim whitespace
            param=$(echo "$param" | xargs)
            if [[ -n "$param" ]]; then
                param_args+=("--pac-param" "$param")
                param_count=$((param_count + 1))
                if [[ "$VERBOSE" == "true" ]]; then
                    log_info "Added PaC parameter: --pac-param $param"
                fi
            fi
        done <<< "$TEKTOR_PAC_PARAMS"
    fi

    if [[ "$VERBOSE" == "true" && $param_count -gt 0 ]]; then
        log_info "Total parameters: $param_count (${#param_args[@]} array elements)"
        log_info "Parameter array: ${param_args[*]}"
    fi

    # If TEKTOR_TASK_DIR is provided, add it to args
    if [[ -n "${TEKTOR_TASK_DIR:-}" ]]; then
        param_args+=("--task-dir" "${TEKTOR_TASK_DIR}")
        if [[ "$VERBOSE" == "true" ]]; then
            log_info "Using --task-dir ${TEKTOR_TASK_DIR}"
        fi
    fi

    # Execute tektor with parameters and capture output
    local tektor_output
    local tektor_exit_code

    if [[ ${#param_args[@]} -gt 0 ]]; then
        tektor_output=$("${TEKTOR_BINARY}" validate "${param_args[@]}" "${file}" 2>&1)
        tektor_exit_code=$?
    else
        # No parameters, use original command
        tektor_output=$("$TEKTOR_BINARY" validate "$file" 2>&1)
        tektor_exit_code=$?
    fi

    # Check if the file was skipped (not a Tekton resource) - exit code 2
    if [[ $tektor_exit_code -eq 2 ]]; then
        log_skipped "? $file (not a Tekton resource)"
        SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
        return 0
    elif [[ $tektor_exit_code -eq 0 ]]; then
        log_success "✓ $file"
        VALIDATED_COUNT=$((VALIDATED_COUNT + 1))
        return 0
    else
        log_error "✗ $file"
        # Show the error output if verbose mode is enabled
        if [[ "$VERBOSE" == "true" && -n "$tektor_output" ]]; then
            echo "$tektor_output"
        fi
        ERROR_COUNT=$((ERROR_COUNT + 1))
        return 1
    fi
}

# Main validation logic
main() {
    log_info "Starting Tekton resource validation..."

    # Check if CHANGED_FILES is set
    if [[ -z "$CHANGED_FILES" ]]; then
        log_error "CHANGED_FILES environment variable is not set"
        exit 1
    fi

    # Check if tektor binary exists
    TEKTOR_BINARY="${GITHUB_ACTION_PATH:-./}/build/tektor"
    if [[ ! -f "$TEKTOR_BINARY" ]]; then
        log_error "tektor binary not found at $TEKTOR_BINARY"
        exit 1
    fi

    # Convert space-separated CHANGED_FILES to array
    read -ra files <<< "$CHANGED_FILES"

    log_info "Found ${#files[@]} changed files to examine"

    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Changed files: ${files[*]}"
    fi

    # Filter and validate Tekton resources
    local validation_failed=false
    for file in "${files[@]}"; do
        CHECKED_COUNT=$((CHECKED_COUNT + 1))

        if [[ "$VERBOSE" == "true" ]]; then
            log_info "Checking file: $file"
        fi

        if ! validate_file "$file"; then
            validation_failed=true
        fi
    done

    # Output summary
    echo ""
    log_info "=== Validation Summary ==="
    log_info "Files checked: $CHECKED_COUNT"
    log_info "Tekton resources validated: $VALIDATED_COUNT"
    log_info "Files skipped (not Tekton resources): $SKIPPED_COUNT"

    if [[ $ERROR_COUNT -eq 0 ]]; then
        log_success "All validations passed! ✓"
    else
        log_error "Found $ERROR_COUNT validation error(s) ✗"
    fi

    # Set outputs for GitHub Actions
    if [[ -n "$GITHUB_OUTPUT" ]]; then
        echo "validated-files=$VALIDATED_COUNT" >> "$GITHUB_OUTPUT"
        echo "validation-errors=$ERROR_COUNT" >> "$GITHUB_OUTPUT"
    fi

    # Exit with error if there were validation failures and fail-on-error is true
    if [[ "$validation_failed" == "true" && "$FAIL_ON_ERROR" == "true" ]]; then
        log_error "Validation failed. Set fail-on-error to false to continue on validation errors."
        exit 1
    fi

    log_success "Validation completed successfully"
}

# Run main function
main "$@"
