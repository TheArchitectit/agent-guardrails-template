# Spec Delta

## Purpose

Defines dated advisory review evidence for JavaScript extension dependency trees, distinguishing runtime dependencies from development tooling.

## ADDED Requirements

### Requirement: Dependency advisory results are dated and scoped

Maintainers SHALL record JavaScript advisory results with scan date, lockfile identity, and runtime versus development dependency scope.

#### Scenario: review extension dependency findings

- **WHEN** a maintainer reviews an extension dependency audit
- **THEN** the record identifies the scanned lockfile and date and reports runtime and development results separately without asserting reachability from counts alone

### Requirement: Advisory results receive an explicit disposition

Maintainers SHALL record a disposition for each dated advisory review before using it as release evidence.

#### Scenario: use an audit as release evidence

- **WHEN** a release review cites a JavaScript dependency audit
- **THEN** the cited record includes a maintainer disposition and its scan context
