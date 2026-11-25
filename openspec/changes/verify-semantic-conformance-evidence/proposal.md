# Proposal: Verify semantic conformance evidence

## Why

The default semantic corpus uses category marker text and a reference classifier that recognizes those markers. It demonstrates harness behavior, not category performance for a configured production classifier.

## What Changes

- Require pinned, representative category positives and benign negatives to be evaluated with the configured classifier.
- Require every shipped category to declare and meet a minimum detection rate and maximum false-positive rate; record both thresholds and measured results.
- Scope reference-classifier runs to harness validation.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `semantic-filtering-evidence-and-conformance`: require representative corpus results from configured classifiers.

## Impact

Semantic corpus fixtures, classifier adapters, and conformance reporting. Release gate wiring and report freshness are covered separately.
