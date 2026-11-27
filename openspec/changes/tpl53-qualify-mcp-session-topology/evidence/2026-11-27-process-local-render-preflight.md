# TPL-53 task 1.2 — rendered process-local profile preflight

The laptop passed the intermediate render and file validator. Its final image
lacked PyYAML. The UCS03 results below cover the original candidate and the
subsequent routing corrections. All commands used synthetic files.

## Supported fixture

Profile: `mcp-server/deploy/profiles/process-local-single-instance.yaml`

The profile declares `session_mode: process-local`, one Deployment replica,
`Recreate`, ClusterIP routing with no session affinity, and HPA bounds 1..1.
The renderer adds the mode annotation as deployment metadata to the actual
Deployment rendered from `mcp-server/deploy/k8s-deployment.yaml`; it does not
invent an application configuration knob, hand-edit, or merely inspect the
source template.

## Lab reproduction

```text
python scripts/render_k8s_profile.py --template mcp-server/deploy/k8s-deployment.yaml --profile mcp-server/deploy/profiles/process-local-single-instance.yaml --version fixture --build-time 1970-01-01T00:00:00Z --git-commit fixture --output artifacts/k8s/process-local-single-instance.yaml
preflight PASS: artifacts/k8s/process-local-single-instance.yaml (process-local, replicas=1, HPA=1..1, strategy=Recreate)
```

The output is deterministic for the pinned values above and contains no
unresolved `${...}` placeholders. The command writes YAML only; no apply path
exists in the tool.

An already-rendered file can be checked independently without rendering or
contacting a cluster:

```text
python scripts/render_k8s_profile.py --validate artifacts/k8s/process-local-single-instance.yaml --profile mcp-server/deploy/profiles/process-local-single-instance.yaml
validation PASS: artifacts/k8s/process-local-single-instance.yaml
```

## Negative fixtures

At `0707208b86284ab96b6464a5c43b1c273d34107b`, the UCS03 lab rendered and
validated the fixture with PyYAML 6.0.1. Its SHA-256 was
`a844757301aedd92db85ef8537aeb3c916089eaf4b9540d505a5cacf6d597db8`.
All seven fixture tests and all 73 strict OpenSpec 2.2.0 items passed.
An invalid replica count preserved an existing output byte for byte. The CLI
rejected malformed YAML, a mismatched namespace and an unknown target port.
It accepted a ClusterIP Service with its entire ports list missing. The
[Regression Guard job](https://github.com/TheArchitectit/agent-guardrails-template/actions/runs/46022116381/job/140584100131)
also required a new test under `tests/regression/`.

The regression at `tests/regression/test_process_local_service_ports.py`
exercises both CLI entry points with a missing, empty or mapping-valued ports
field and checks that a rejected render preserves the previous output.
The serialized validator also accepted an invalid two-replica profile alongside
a valid manifest. A fixture now varies the requested replica count, HPA,
rollout strategy and affinity while holding the rendered file constant.

`tests/test_render_k8s_profile.py` mutates the rendered fixture and verifies a
`PreflightError` for each of the following before output is written:

* replicas greater than one;
* HPA `maxReplicas` greater than one;
* `RollingUpdate` / overlapping revisions; and
* a pod annotation that disagrees with the declared `process-local` profile.

The preflight also rejects a missing or mismatched Service selector, a
non-ClusterIP Service, HPA target mismatch, unresolved placeholders, and any
unsupported session mode.

## Boundaries left open

This receipt does not claim restart/reinitialization, revocation, principal
binding, multi-pod routing, or a live deployment. Those remain task 1.3 and
the independent review in task 2.1.
