# postgres-connection-credential-encoding Specification

## Purpose

Specify safe construction of PostgreSQL connection URLs when credentials contain URI-reserved characters.

## Requirements

### Requirement: PostgreSQL URL credentials are encoded as URI userinfo
The configuration layer SHALL encode the PostgreSQL username and password as URL userinfo before constructing the connection URL, while preserving the established host, database path, SSL mode, and timeout parameters.

#### Scenario: Ordinary credentials keep the established URL
- **GIVEN** the PostgreSQL username and password contain only ordinary unreserved characters
- **WHEN** the configuration layer builds its connection URL
- **THEN** it returns the existing URL format and query parameter order

#### Scenario: Reserved characters round-trip through URL parsing
- **GIVEN** the PostgreSQL password contains URI-reserved punctuation
- **WHEN** the configuration layer builds its connection URL and a PostgreSQL URL parser reads it
- **THEN** parsing succeeds and recovers the original username and password exactly
