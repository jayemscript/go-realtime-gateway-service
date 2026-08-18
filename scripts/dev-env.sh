#!/usr/bin/env bash

quiet=false
if [[ "${1:-}" == "--quiet" ]]; then
    quiet=true
fi

if ! command -v go >/dev/null 2>&1; then
    go_candidates=(
        "/c/Program Files/Go/bin/go.exe"
        "/usr/local/go/bin/go"
        "/usr/bin/go"
    )

    for go_executable in "${go_candidates[@]}"; do
        if [[ -x "$go_executable" ]]; then
            go_bin="$(dirname -- "$go_executable")"
            export PATH="$go_bin:$PATH"
            break
        fi
    done
fi

if ! command -v go >/dev/null 2>&1; then
    echo "Go was not found on PATH or in a standard installation directory." >&2
    echo "Install Go, open a new Bash session, and run the preflight again." >&2
    return 1 2>/dev/null || exit 1
fi

if [[ "$quiet" == false ]]; then
    printf 'Go environment ready: %s\n' "$(command -v go)"
    go version
fi
