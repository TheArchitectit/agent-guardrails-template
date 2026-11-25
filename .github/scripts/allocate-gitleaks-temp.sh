#!/usr/bin/env bash
set -euo pipefail

runner_temp=${RUNNER_TEMP:?RUNNER_TEMP must be set by GitHub Actions}
github_output=${GITHUB_OUTPUT:?GITHUB_OUTPUT must be set by GitHub Actions}
mkdir -p -- "$runner_temp"
runner_temp=$(cd -- "$runner_temp" && pwd -P)

install_dir=
cleanup_failed_allocation() {
  status=$?
  if [[ "$status" != 0 && -n "$install_dir" ]]; then
    rm -rf -- "$install_dir"
  fi
  return "$status"
}
trap cleanup_failed_allocation EXIT

install_dir=$(mktemp -d "$runner_temp/gitleaks.XXXXXXXX")
printf 'install_dir=%s\n' "$install_dir" >> "$github_output"
