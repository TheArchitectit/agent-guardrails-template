#!/usr/bin/env python3
"""Render and preflight the repository Kubernetes deployment profiles.

This command only renders YAML and validates the rendered objects. It never
contacts a cluster and deliberately has no apply operation.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path
from typing import Any

import yaml


PLACEHOLDER = re.compile(r"\$\{[^}]+\}")
SESSION_ANNOTATION = "guardrails.dev/session-mode"


class PreflightError(ValueError):
    """A rendered profile is not safe for its declared session mode."""


def _find(documents: list[dict[str, Any]], kind: str) -> list[dict[str, Any]]:
    return [doc for doc in documents if doc.get("kind") == kind]


def _replace(value: Any, values: dict[str, str]) -> Any:
    if isinstance(value, str):
        def replacement(match: re.Match[str]) -> str:
            expression = match.group(0)[2:-1]
            name, _, default = expression.partition(":-")
            if name in values:
                return values[name]
            if default:
                return default
            raise PreflightError(f"missing render value: {name}")

        return PLACEHOLDER.sub(replacement, value)
    if isinstance(value, list):
        return [_replace(item, values) for item in value]
    if isinstance(value, dict):
        return {key: _replace(item, values) for key, item in value.items()}
    return value


def _container_env(deployment: dict[str, Any]) -> dict[str, str]:
    containers = deployment.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [])
    if len(containers) != 1:
        raise PreflightError("deployment must contain exactly one application container")
    env = {}
    for item in containers[0].get("env", []):
        if "name" in item and "value" in item:
            env[item["name"]] = str(item["value"])
    return env


def preflight(documents: list[dict[str, Any]], profile: dict[str, Any]) -> None:
    mode = profile.get("session_mode")
    if mode != "process-local":
        raise PreflightError(f"unsupported session_mode: {mode!r}")
    deployments = _find(documents, "Deployment")
    services = _find(documents, "Service")
    hpas = _find(documents, "HorizontalPodAutoscaler")
    if len(deployments) != 1:
        raise PreflightError("profile must render exactly one Deployment")
    if len(services) != 1:
        raise PreflightError("profile must render exactly one Service")
    if len(hpas) != 1:
        raise PreflightError("process-local profile must render exactly one HPA pinned to one replica")

    deployment = deployments[0]
    spec = deployment.get("spec", {})
    replicas = spec.get("replicas")
    if replicas != 1:
        raise PreflightError(f"process-local requires replicas=1; rendered replicas={replicas!r}")
    strategy = spec.get("strategy", {})
    if strategy.get("type") != "Recreate":
        raise PreflightError("process-local requires strategy.type=Recreate; overlapping revisions are unsafe")
    if "rollingUpdate" in strategy:
        raise PreflightError("process-local must not render rollingUpdate or overlapping revisions")

    annotations = spec.get("template", {}).get("metadata", {}).get("annotations", {})
    env = _container_env(deployment)
    declared_mode = annotations.get(SESSION_ANNOTATION) or env.get("MCP_SESSION_MODE")
    if declared_mode != mode or annotations.get(SESSION_ANNOTATION) != mode or env.get("MCP_SESSION_MODE") != mode:
        raise PreflightError("session mode must be declared consistently in pod annotation and MCP_SESSION_MODE")

    service = services[0].get("spec", {})
    if service.get("sessionAffinity", "None") != profile.get("service_session_affinity", "None"):
        raise PreflightError("Service sessionAffinity conflicts with process-local routing contract")
    if profile.get("routing") != "cluster-ip" or service.get("type") != "ClusterIP":
        raise PreflightError("process-local routing must use a ClusterIP Service")

    hpa_spec = hpas[0].get("spec", {})
    if hpa_spec.get("minReplicas") != 1 or hpa_spec.get("maxReplicas") != 1:
        raise PreflightError("process-local HPA must have minReplicas=maxReplicas=1")
    if hpa_spec.get("scaleTargetRef", {}).get("name") != deployment.get("metadata", {}).get("name"):
        raise PreflightError("HPA target does not match rendered Deployment")

    selector = spec.get("selector", {}).get("matchLabels", {})
    pod_labels = spec.get("template", {}).get("metadata", {}).get("labels", {})
    service_selector = service.get("selector", {})
    if not selector or any(pod_labels.get(k) != v for k, v in selector.items()):
        raise PreflightError("Deployment selector does not match its pod labels")
    if service_selector != selector:
        raise PreflightError("Service selector does not match Deployment selector")


def render(template: Path, profile_path: Path, values: dict[str, str]) -> list[dict[str, Any]]:
    profile = yaml.safe_load(profile_path.read_text(encoding="utf-8"))
    if not isinstance(profile, dict):
        raise PreflightError("profile must be a YAML mapping")
    raw = template.read_text(encoding="utf-8")
    documents = [_replace(doc, values) for doc in yaml.safe_load_all(raw)]
    deployment = next((doc for doc in documents if doc and doc.get("kind") == "Deployment"), None)
    if deployment is None:
        raise PreflightError("template has no Deployment")
    deployment["spec"]["replicas"] = profile["replicas"]
    deployment["spec"]["strategy"] = {"type": profile["strategy"]}
    metadata = deployment["spec"]["template"].setdefault("metadata", {})
    metadata.setdefault("annotations", {})[SESSION_ANNOTATION] = profile["session_mode"]
    containers = deployment["spec"]["template"]["spec"]["containers"]
    env = containers[0].setdefault("env", [])
    existing = next((item for item in env if item.get("name") == "MCP_SESSION_MODE"), None)
    if existing is None:
        env.append({"name": "MCP_SESSION_MODE", "value": profile["session_mode"]})
    else:
        existing["value"] = profile["session_mode"]
    hpa = next((doc for doc in documents if doc and doc.get("kind") == "HorizontalPodAutoscaler"), None)
    if hpa is None:
        raise PreflightError("template has no HorizontalPodAutoscaler")
    hpa["spec"]["minReplicas"] = profile["hpa"]["minReplicas"]
    hpa["spec"]["maxReplicas"] = profile["hpa"]["maxReplicas"]
    return documents


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--template", type=Path, default=Path("mcp-server/deploy/k8s-deployment.yaml"))
    parser.add_argument("--profile", type=Path, default=Path("mcp-server/deploy/profiles/process-local-single-instance.yaml"))
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--version", default="fixture")
    parser.add_argument("--build-time", default="1970-01-01T00:00:00Z")
    parser.add_argument("--git-commit", default="fixture")
    args = parser.parse_args(argv)
    try:
        values = {"VERSION": args.version, "BUILD_TIME": args.build_time, "GIT_COMMIT": args.git_commit}
        documents = render(args.template, args.profile, values)
        profile = yaml.safe_load(args.profile.read_text(encoding="utf-8"))
        preflight(documents, profile)
        output = "---\n".join(yaml.safe_dump(doc, sort_keys=False).rstrip() for doc in documents if doc) + "\n"
        if PLACEHOLDER.search(output):
            raise PreflightError("rendered output contains unresolved placeholders")
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(output, encoding="utf-8", newline="\n")
        print(f"preflight PASS: {args.output} (process-local, replicas=1, HPA=1..1, strategy=Recreate)")
        return 0
    except (OSError, yaml.YAMLError, PreflightError) as exc:
        print(f"preflight FAIL: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
