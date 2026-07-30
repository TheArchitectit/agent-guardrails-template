# Tech Stack Inventory and Standards (Enterprise Baseline)

## Overview
- Targeted stack for the Guardrails Template enterprise program, with multi-cloud readiness and open governance.

## Cloud and platform
- Primary cloud: AWS (multi-region, data residency considerations)
- Multi-cloud compatibility notes (Azure/GCP patterns)

## IaC and configuration
- Terraform + Pulumi (multi-language support)
- GitOps for IaC with plan/apply gates

## CI/CD and code quality
- GitHub Actions as primary CI/CD engine
- Optional Jenkins/GitLab adapters if needed

## Languages and runtimes
- Frontend/UI: TypeScript + React
- Backend: Node.js/TS, Python, Go
- Data: PostgreSQL, Redis (as needed)

## Observability and tracing
- OpenTelemetry for traces; Jaeger/Tempo; Prometheus + Grafana

## Security and governance tooling
- SBOM tooling: Snyk/Dependabot; SPDX with Syft
- Secrets management: HashiCorp Vault / AWS Secrets Manager
- Policy as code: OPA with Rego; Terraform Sentinel

## Documentation tooling
- MkDocs with Material for MkDocs OR Docusaurus
- Linters: markdownlint, Vale
- Doc build QA: dead-link checks, header parity, front-matter checks

## Licensing and procurement
- Licensing posture and SBOM exports per release

Notes
- Tailor to regulatory constraints as needed.
