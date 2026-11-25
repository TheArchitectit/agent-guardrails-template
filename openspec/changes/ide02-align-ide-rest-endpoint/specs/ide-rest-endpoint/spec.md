# Spec Delta

## Purpose

Defines the default REST endpoint used by supported IDE integrations and preserves explicit endpoint overrides.

## ADDED Requirements

### Requirement: IDE clients default to the server REST listener
VS Code, JetBrains, Vim, and Neovim integrations SHALL use `http://localhost:8081` for REST requests when no endpoint override is configured; the separate MCP transport default SHALL remain port 8080.

#### Scenario: fresh IDE configuration
- **GIVEN** an IDE integration has no saved endpoint override
- **WHEN** it requests health or IDE validation from a default local server
- **THEN** it sends the REST request to port 8081 and does not use the MCP transport port

#### Scenario: explicit endpoint override
- **GIVEN** a user configures a non-default server URL
- **WHEN** the IDE sends a REST request
- **THEN** it uses the configured URL instead of replacing it with the default
