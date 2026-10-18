#!/usr/bin/env sh
# Internal Markdown link check for the R19-11 docs gate.
#
# Fails (exit 1) when any relative Markdown link or image points at a path that
# does not exist. External (http/https/mailto) links, pure-anchor links and
# non-file schemes are not checked.
#
# The optional first argument selects the tree to scan (default: ".").
#
# This is the single source of truth for the docs-gate link check: the CI job
# (`documentation-check.yml`, `regression-guard.yml`) and the mutation-kill
# harness (`internal/mutationkill`) both invoke this script, so the control they
# exercise is the same one that blocks a merge.
#
# Scope rules (each one exists because a naive checker got it wrong):
#   1. Fenced code blocks are SKIPPED. A documentation example that shows a
#      link pattern (for example `[x](pattern.md)` inside a ```bash fence) is
#      not a link and must not be reported as a broken one.
#   2. Inline code spans (`...`) are removed before extraction, for the same
#      reason.
#   3. Targets are extracted from the `](target)` suffix rather than by matching
#      a whole `[text](target)` construct. Link text may itself contain
#      brackets (`[x] No .env files (see [y](y.md))`); a construct-level regex
#      silently reports the text as the target.
#   4. A fragment (`#anchor`) and a query (`?x`) are stripped before the
#      existence check; they do not name a different file.
#   5. Non-vacuity: a scan that finds no Markdown file, or no link at all, is a
#      FAILURE. "I matched nothing" is not "everything is correct".
#
# Named limits: reference-style definitions (`[id]: path.md`) and angle-bracket
# destinations with spaces are not parsed; HTML `<a href>` is not parsed.
set -eu

root="${1:-.}"

files=$(find "$root" -name '*.md' -type f 2>/dev/null | grep -v node_modules | grep -v '/\.git/' || true)

if [ -z "$files" ]; then
    echo "DOC LINKS: FAIL — no Markdown files found under '$root' (non-vacuous scan required)"
    exit 1
fi

broken=""
scanned_files=0
scanned_links=0

for file in $files; do
    scanned_files=$((scanned_files + 1))
    dir=$(dirname "$file")

    # Strip fenced blocks and inline code, then emit one target per line.
    targets=$(awk '
        BEGIN { fence = 0 }
        /^[[:space:]]*(```|~~~)/ { fence = !fence; next }
        fence == 1 { next }
        {
            line = $0
            gsub(/`[^`]*`/, "", line)          # drop inline code spans
            while (match(line, /\]\([^)]*\)/)) {
                t = substr(line, RSTART + 2, RLENGTH - 3)
                print t
                line = substr(line, RSTART + RLENGTH)
            }
        }
    ' "$file" 2>/dev/null || true)

    [ -n "$targets" ] || continue

    for link in $targets; do
        # Strip fragment/query before testing the path.
        link_path=$(printf '%s' "$link" | cut -d'#' -f1 | cut -d'?' -f1)

        # Skip external, mail, and pure-anchor links.
        case "$link_path" in
            ''|'#'*) continue ;;
            http://*|https://*|mailto:*|tel:*|ftp://*) continue ;;
        esac

        scanned_links=$((scanned_links + 1))

        # Absolute-from-root targets are checked against the scan root.
        case "$link_path" in
            /*) full_path="$root$link_path" ;;
            *)  full_path="$dir/$link_path" ;;
        esac

        if [ ! -e "$full_path" ]; then
            broken="${broken}${file}: broken link to ${link_path}\n"
        fi
    done
done

if [ "$scanned_files" -eq 0 ]; then
    echo "DOC LINKS: FAIL — scanned no Markdown files (non-vacuous scan required)"
    exit 1
fi

if [ "$scanned_links" -eq 0 ]; then
    echo "DOC LINKS: FAIL — scanned $scanned_files Markdown files but resolved zero internal links; a checker that matches nothing cannot pass"
    exit 1
fi

if [ -n "$broken" ]; then
    echo "=========================================="
    echo "BROKEN LINKS FOUND:"
    printf '%b' "$broken"
    echo "=========================================="
    echo "DOC LINKS: FAIL — $(printf '%b' "$broken" | grep -c 'broken link') broken of $scanned_links internal links across $scanned_files files"
    exit 1
fi

echo "DOC LINKS: OK — $scanned_links internal links across $scanned_files Markdown files resolve"
