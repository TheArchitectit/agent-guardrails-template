# Tasks

## 1. Declare and install Python job dependencies

- [ ] 1.1 Select or add the maintained dependency manifest used by root Python scripts, update GitLab validation/test/export job installs, and verify each referenced file exists in a clean checkout.
- [ ] 1.2 Run the validation, Python test/coverage, and export job commands in a clean environment and verify expected artifacts and exit statuses.

## 2. Preserve pipeline behavior

- [ ] 2.1 Verify merge-request/default-branch rules, coverage report path, and export artifact remain configured after the dependency update.
