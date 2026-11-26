# Tasks

## 1. Package and resolve backend

- [x] 1.1 Add the backend and required runtime files to the Makefile install layout, update CLI resolution relative to the installation, and verify a command works from a temporary install prefix outside the checkout.
- [x] 1.2 Update `install.sh` to install the same backend payload and verify its installed command works from an unrelated directory.

## 2. Preserve override and uninstall behavior

- [x] 2.1 Add temporary-prefix tests for `TEAM_MANAGER_PATH` precedence and uninstall ownership, then verify user override files remain untouched.
