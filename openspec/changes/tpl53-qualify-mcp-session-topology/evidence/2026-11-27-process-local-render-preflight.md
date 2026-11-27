# TPL-53 task 1.2 — rendered process-local profile preflight

This receipt covers render/preflight only. It does not contact Kubernetes and
does not run `kubectl apply`, a cluster drill, a restart, or an application
pilot.

## Supported fixture

Profile: `mcp-server/deploy/profiles/process-local-single-instance.yaml`

The profile declares `session_mode: process-local`, one Deployment replica,
`Recreate`, ClusterIP routing with no session affinity, and HPA bounds 1..1.
The renderer adds the mode annotation and `MCP_SESSION_MODE` to the actual
Deployment rendered from `mcp-server/deploy/k8s-deployment.yaml`; it does not
hand-edit or merely inspect the source template.

## Runnable receipt

```text
python scripts/render_k8s_profile.py --template mcp-server/deploy/k8s-deployment.yaml --profile mcp-server/deploy/profiles/process-local-single-instance.yaml --version fixture --build-time 1970-01-01T00:00:00Z --git-commit fixture --output artifacts/k8s/process-local-single-instance.yaml
preflight PASS: artifacts/k8s/process-local-single-instance.yaml (process-local, replicas=1, HPA=1..1, strategy=Recreate)
```

The output is deterministic for the pinned values above and contains no
unresolved `${...}` placeholders. The command writes YAML only; no apply path
exists in the tool.

## Negative coverage

`tests/test_render_k8s_profile.py` mutates the rendered fixture and verifies a
`PreflightError` for each of the following before output is written:

* replicas greater than one;
* HPA `maxReplicas` greater than one;
* `RollingUpdate` / overlapping revisions; and
* a pod annotation that disagrees with `MCP_SESSION_MODE`.

The preflight also rejects a missing or mismatched Service selector, a
non-ClusterIP Service, HPA target mismatch, unresolved placeholders, and any
unsupported session mode.

## Boundaries left open

This receipt does not claim restart/reinitialization, revocation, principal
binding, multi-pod routing, or a live deployment. Those remain task 1.3 and
the independent review in task 2.1.
