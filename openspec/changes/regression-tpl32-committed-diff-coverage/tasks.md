# Tasks

## 1. Add committed-range evaluation

- [ ] 1.1 Add explicit base/head arguments to the guard and use the event's base-to-head patch and matching patch content; reject missing/unresolvable bases, incomplete history, or empty CI ranges. Preserve local staged/unstaged options.
- [ ] 1.2 Add a temporary-Git-repository test with a benign added sentinel that matches a temporary registry rule; assert the base/head path reports it, an unchanged range passes, and unresolved/empty bases fail closed. Verify with `python3 -m pytest tests/test_regression_check.py -q`.

## 2. Wire event ranges and retain local behavior

- [ ] 2.1 Update `.github/workflows/regression-guard.yml` to pass PR base/head or push before/head SHAs and retain `fetch-depth: 0`; verify each event has a nonempty range.
- [ ] 2.2 Run `python3 -m pytest tests/test_regression_check.py -q` and exercise the CI range command in a temporary fixture checkout. Include staged-only, unstaged-only, and mixed edits with known registry/pattern matches; verify `--all` inspects both patch contents and existing local flags still work.
