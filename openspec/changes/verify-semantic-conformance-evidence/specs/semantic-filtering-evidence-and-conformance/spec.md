# Semantic Filtering Evidence and Conformance

## MODIFIED Requirements

### Requirement: per-category conformance corpora

Each shipped category SHALL have versioned, representative positive and benign negative fixtures. It SHALL declare a minimum detection rate and maximum false-positive rate. Conformance SHALL evaluate the configured classifier against both thresholds and SHALL report measured rates, versions, and missed fixtures. A reference classifier MAY validate harness mechanics but SHALL NOT be represented as production classifier performance.

#### Scenario: category misses detection bar

- **GIVEN** a category whose pinned corpus detection rate falls below its declared minimum or whose false-positive rate exceeds its declared maximum
- **WHEN** conformance runs with the configured classifier
- **THEN** that category's verified claim is withheld and the report names the failing fixtures and measured results

#### Scenario: reference classifier validates harness

- **GIVEN** conformance uses a marker-recognition reference classifier
- **WHEN** its results are reported
- **THEN** the report labels them as harness validation and does not claim production classifier performance
