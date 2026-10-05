#!/usr/bin/env sh
# Internal Markdown link check for the R19-11 docs gate.
#
# Fails (exit 1) when any relative Markdown link points at a file that does not
# exist. External (http/https) links and pure-anchor links are not checked.
# The optional first argument selects the tree to scan (default: ".").
#
# This is the single source of truth for the docs-gate link check: the CI job
# (`.github/workflows/documentation-check.yml`) and the mutation-kill harness
# (`internal/mutationkill`) both invoke this script so the control they exercise
# is the same one that blocks a merge.
set -eu

root="${1:-.}"

broken=""
files=$(find "$root" -name '*.md' -type f 2>/dev/null | grep -v node_modules | grep -v '/\.git/' || true)

for file in $files; do
    # Extract relative .md links: [text](target.md) / [text](target.md#anchor).
    links=$(grep -oE '\[.*\]\([^)]+\.md[^)]*\)' "$file" 2>/dev/null | grep -oE '\([^)]+\)' | tr -d '()' || true)

    for link in $links; do
        link_path=$(printf '%s' "$link" | cut -d'#' -f1)

        # Skip external links.
        case "$link_path" in
            http*) continue ;;
        esac

        dir=$(dirname "$file")
        full_path="$dir/$link_path"

        if [ ! -f "$full_path" ]; then
            broken="${broken}${file}: broken link to ${link_path}\n"
        fi
    done
done

if [ -n "$broken" ]; then
    echo "=========================================="
    echo "BROKEN LINKS FOUND:"
    printf '%b' "$broken"
    exit 1
fi

echo "All internal links are valid."
