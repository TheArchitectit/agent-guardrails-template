# Tasks

## 1. Establish dated audit evidence

- [x] 1.1 Add a repeatable lockfile-backed audit procedure for Pi and VS Code and verify it records date, lockfile identity, tool version, and separate `npm audit` and `npm audit --omit=dev` results.
- [ ] 1.2 Record a maintainer disposition for each reported package entry and verify that the record distinguishes development-only results from runtime results without inferred reachability.

## 2. Integrate review

- [x] 2.1 Add the dated audit record to the release review process and verify a release checklist links to the record and captures disposition.
