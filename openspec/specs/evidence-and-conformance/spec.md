# evidence-and-conformance Specification

Status: CLOSED (EXERCISED 6)

## Purpose

Bind semantic filtering decisions to auditable redaction, prove each category against its own conformance corpus, and invalidate the evidence when the detector version changes.

## Requirements

### Requirement: per-decision evidence

Every classification and mediation decision SHALL emit an AIGGP envelope carrying subject identity, classifier identity and version, bundle digest, and raw classification output.

#### Scenario: classifier swap invalidates claims

Given a module release that changes classifier version without re-running the corpus, when conformance status is queried, then prior pass claims SHALL be reported as stale.

### Requirement: adversarial conformance corpus

A versioned public corpus SHALL cover at minimum: direct injection, instruction-in-content, encoded/obfuscated payloads, multi-turn setup, authority impersonation, and exfiltration lures. Verified status SHALL require meeting the declared pass bar per category on the pinned corpus version.

#### Scenario: category miss published

Given a corpus category below the bar, when the conformance report is generated, then the category SHALL be listed as failing with fixtures named.

### Requirement: benign-corpus false-positive budget

The module SHALL declare and meet a false-positive budget on a benign corpus. Regression beyond budget SHALL block release.

#### Scenario: budget breach

Given a release candidate exceeding the budget, when the release gate runs, then it SHALL FAIL with the measured rate.

### Requirement: auditable redaction

Redacted content SHALL be recorded as a content hash plus category metadata in the evidence envelope. Raw sensitive payloads SHALL NOT be written to the ledger.

#### Scenario: secret redaction

Given a commit containing a credential, when filtering redacts it, then the envelope SHALL carry the hash and category, and the ledger SHALL contain no plaintext secret.

### Requirement: per-category conformance corpora

Each category SHALL have a versioned detection corpus and benign corpus. Verified status SHALL require meeting declared detection and false-positive budgets on the pinned corpus versions.

#### Scenario: false-positive budget breach

Given a category exceeding its benign-corpus budget, when the release gate runs, then it SHALL FAIL with measured rates.

### Requirement: version-bump invalidation

A classifier version change SHALL invalidate prior conformance claims until the corpus is re-run at the new version.

#### Scenario: silent upgrade flagged

Given a deployment whose classifier updated without a corpus re-run, when conformance status is queried, then the category SHALL report stale.
