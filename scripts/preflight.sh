#!/usr/bin/env bash

set -uo pipefail

expected_go_version="go version go1.26.6 windows/amd64"
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "$script_dir/.." && pwd -P)"
failures=()

pass_check() {
    local name="$1"
    local details="$2"
    printf '[PASS] %s - %s\n' "$name" "$details"
}

fail_check() {
    local name="$1"
    local details="$2"
    failures+=("$name - $details")
    printf '[FAIL] %s - %s\n' "$name" "$details" >&2
}

run_command() {
    local name="$1"
    local check_output
    local command_status
    shift

    check_output="$("$@" 2>&1)"
    command_status=$?

    if (( command_status == 0 )); then
        pass_check "$name" "$check_output"
        return 0
    fi

    if [[ -z "$check_output" ]]; then
        check_output="command exited with code $command_status"
    fi
    fail_check "$name" "$check_output"
    return 1
}

cd -- "$repository_root" || exit 1

if source "$script_dir/dev-env.sh" --quiet; then
    go_version="$(go version 2>&1)"
    if [[ "$go_version" == "$expected_go_version" ]]; then
        pass_check "Go" "$go_version"
    else
        fail_check "Go" "expected '$expected_go_version', received '$go_version'"
    fi
else
    fail_check "Go" "unable to configure the Go toolchain"
fi

run_command "Docker client" docker --version || true
run_command "Docker daemon" docker version --format '{{.Server.Version}}' || true
run_command "Docker Compose" docker compose version --short || true

if git_root="$(git rev-parse --show-toplevel 2>&1)"; then
    resolved_git_root="$(cd -- "$git_root" && pwd -P)"
    if [[ "$resolved_git_root" == "$repository_root" ]]; then
        pass_check "Git repository" "$resolved_git_root"
    else
        fail_check "Git repository" "expected '$repository_root', received '$resolved_git_root'"
    fi
else
    fail_check "Git repository" "$git_root"
fi

if git_status="$(git status --short --branch 2>&1)"; then
    status_heading="${git_status%%$'\n'*}"
    pass_check "Git status" "$status_heading"
    printf '%s\n' "$git_status"
else
    fail_check "Git status" "$git_status"
fi

forbidden_paths=()
if repository_inventory="$({
    git ls-files --cached
    git ls-files --others --exclude-standard
    git ls-files --others --ignored --exclude-standard
} 2>&1 | sort -u)"; then
    while IFS= read -r inventory_path; do
        [[ -z "$inventory_path" ]] && continue

        repository_path="${inventory_path//\\//}"
        path_with_boundaries="/$repository_path/"

        case "$path_with_boundaries" in
            */.pnpm-store/* | */node_modules/* | */vendor/* | */.cache/* | */tmp/* | */temp/* | */dist/* | */build/* | */coverage/*)
                forbidden_paths+=("$repository_path")
                continue
                ;;
        esac

        case "$repository_path" in
            .env | */.env | .env.* | */.env.*)
                if [[ "${repository_path##*/}" != ".env.example" ]]; then
                    forbidden_paths+=("$repository_path")
                fi
                ;;
            coverage.out | */coverage.out | *.log | *.exe | *.dll | *.test | *.tmp | *.pem | *.key)
                forbidden_paths+=("$repository_path")
                ;;
        esac
    done <<< "$repository_inventory"

    if (( ${#forbidden_paths[@]} == 0 )); then
        pass_check "Repository hygiene" "no forbidden caches, artifacts, local secrets, or temporary files found"
    else
        fail_check "Repository hygiene" "${forbidden_paths[*]}"
    fi
else
    fail_check "Repository inventory" "$repository_inventory"
fi

if (( ${#failures[@]} > 0 )); then
    printf '\nPhase 0 preflight failed with %d problem(s):\n' "${#failures[@]}" >&2
    for failure in "${failures[@]}"; do
        printf -- '- %s\n' "$failure" >&2
    done
    exit 1
fi

printf '\nPhase 0 preflight passed.\n'
