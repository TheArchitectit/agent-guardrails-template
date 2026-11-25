#!/usr/bin/env bash
set -euo pipefail

runner_temp=${RUNNER_TEMP:?RUNNER_TEMP must be set by GitHub Actions}
install_dir=${GITLEAKS_TEMP_DIR:?GITLEAKS_TEMP_DIR must be set by workflow}
runner_temp=$(cd -- "$runner_temp" && pwd -P)

[[ -n "$install_dir" && ! -L "$install_dir" ]] || {
  echo "Refusing to clean an invalid Gitleaks temporary path" >&2
  exit 1
}
[[ "$(dirname -- "$install_dir")" == "$runner_temp" && "$(basename -- "$install_dir")" == gitleaks.* ]] || {
  echo "Refusing to clean a path outside the job-owned Gitleaks temporary directory" >&2
  exit 1
}
rm -rf -- "$install_dir"
echo "Removed this job's Gitleaks temporary directory"
