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
        if not isinstance(item, dict) or not isinstance(item.get("name"), str):
            raise PreflightError("every container env entry must have a string name")
        if item["name"] in env:
            raise PreflightError(f"duplicate container environment variable: {item['name']}")
        if ("value" in item) == ("valueFrom" in item):
            raise PreflightError(f"env {item['name']} must contain exactly one of value or valueFrom")
        if "value" in item:
            env[item["name"]] = str(item["value"])
    return env


def _profile(profile: Any) -> dict[str, Any]:
    if not isinstance(profile, dict):
        raise PreflightError("profile must be a YAML mapping")
    required = ("name", "session_mode", "replicas", "strategy", "service_session_affinity", "hpa", "routing")
    missing = [key for key in required if key not in profile]
    if missing:
        raise PreflightError(f"profile missing required keys: {', '.join(missing)}")
    if not isinstance(profile["replicas"], int) or isinstance(profile["replicas"], bool):
        raise PreflightError("profile replicas must be an integer, not boolean or string")
    hpa = profile["hpa"]
    if not isinstance(hpa, dict) or not isinstance(hpa.get("minReplicas"), int) or isinstance(hpa.get("minReplicas"), bool) or not isinstance(hpa.get("maxReplicas"), int) or isinstance(hpa.get("maxReplicas"), bool):
        raise PreflightError("profile HPA minReplicas and maxReplicas must be integers")
    if profile["session_mode"] == "process-local":
        if profile["replicas"] != 1:
            raise PreflightError("process-local profile requires replicas=1")
        if (hpa["minReplicas"], hpa["maxReplicas"]) != (1, 1):
            raise PreflightError("process-local profile requires HPA minReplicas=maxReplicas=1")
        if profile["strategy"] != "Recreate":
            raise PreflightError("process-local profile requires strategy.type=Recreate")
        if profile["routing"] != "cluster-ip" or profile["service_session_affinity"] != "None":
            raise PreflightError("process-local profile requires ClusterIP routing with no session affinity")
    return profile


def _port_map(deployment: dict[str, Any]) -> tuple[dict[str, int], set[int]]:
    containers = deployment.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [])
    named, numeric = {}, set()
    for container in containers:
        for port in container.get("ports", []):
            if not isinstance(port, dict) or not isinstance(port.get("containerPort"), int) or isinstance(port.get("containerPort"), bool):
                raise PreflightError("container ports must use integer containerPort values")
            numeric.add(port["containerPort"])
            if "name" in port:
                if not isinstance(port["name"], str) or port["name"] in named:
                    raise PreflightError("container port names must be unique strings")
                named[port["name"]] = port["containerPort"]
    return named, numeric


def _validate_service_ports(service: dict[str, Any], deployment: dict[str, Any]) -> None:
    ports = service.get("spec", {}).get("ports")
    if not isinstance(ports, list) or not ports:
        raise PreflightError("Service ports must be a non-empty list for ClusterIP routing")
    named, numeric = _port_map(deployment)
    for port in ports:
        if not isinstance(port, dict) or not isinstance(port.get("port"), int) or isinstance(port.get("port"), bool):
            raise PreflightError("Service ports must use integer port values")
        target = port.get("targetPort")
        if isinstance(target, bool) or (isinstance(target, int) and target not in numeric) or (isinstance(target, str) and target not in named):
            raise PreflightError(f"Service targetPort {target!r} does not match a rendered container port")
        if not isinstance(target, (int, str)):
            raise PreflightError("Service targetPort must be an integer or named container port")


def preflight(documents: list[dict[str, Any]], profile: dict[str, Any]) -> None:
    profile = _profile(profile)
    seen: set[tuple[str, str, str]] = set()
    for doc in documents:
        if not isinstance(doc, dict) or not isinstance(doc.get("apiVersion"), str) or not isinstance(doc.get("kind"), str) or not isinstance(doc.get("metadata"), dict):
            raise PreflightError("every rendered document must have apiVersion, kind, and metadata")
        metadata = doc["metadata"]
        if not isinstance(metadata.get("name"), str) or not isinstance(metadata.get("namespace"), str):
            raise PreflightError("every rendered document must have string metadata.name and metadata.namespace")
        identity = (doc["apiVersion"], doc["kind"], f"{metadata['namespace']}/{metadata['name']}")
        if identity in seen:
            raise PreflightError(f"duplicate rendered object: {identity[1]} {identity[2]}")
        seen.add(identity)
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
    if not isinstance(replicas, int) or isinstance(replicas, bool) or replicas != 1:
        raise PreflightError(f"process-local requires replicas=1; rendered replicas={replicas!r}")
    strategy = spec.get("strategy", {})
    if strategy.get("type") != "Recreate":
        raise PreflightError("process-local requires strategy.type=Recreate; overlapping revisions are unsafe")
    if "rollingUpdate" in strategy:
        raise PreflightError("process-local must not render rollingUpdate or overlapping revisions")

    annotations = spec.get("template", {}).get("metadata", {}).get("annotations", {})
    if annotations.get(SESSION_ANNOTATION) != mode:
        raise PreflightError("session mode must be declared in the Deployment pod annotation")
    _container_env(deployment)

    service = services[0].get("spec", {})
    if service.get("sessionAffinity", "None") != profile.get("service_session_affinity", "None"):
        raise PreflightError("Service sessionAffinity conflicts with process-local routing contract")
    if profile.get("routing") != "cluster-ip" or service.get("type") != "ClusterIP":
        raise PreflightError("process-local routing must use a ClusterIP Service")
    _validate_service_ports(services[0], deployment)

    deployment_meta = deployment.get("metadata", {})
    service_meta = services[0].get("metadata", {})
    hpa_meta = hpas[0].get("metadata", {})
    if deployment_meta.get("namespace") != service_meta.get("namespace") or deployment_meta.get("namespace") != hpa_meta.get("namespace"):
        raise PreflightError("Deployment, Service, and HPA namespaces must match")
    if deployment_meta.get("name") != service_meta.get("name") or deployment_meta.get("name") != hpa_meta.get("name"):
        raise PreflightError("Deployment, Service, and HPA names must match")

    hpa_spec = hpas[0].get("spec", {})
    if any(not isinstance(hpa_spec.get(key), int) or isinstance(hpa_spec.get(key), bool) or hpa_spec.get(key) != 1 for key in ("minReplicas", "maxReplicas")):
        raise PreflightError("process-local HPA must have minReplicas=maxReplicas=1")
    target = hpa_spec.get("scaleTargetRef", {})
    if target.get("apiVersion") != "apps/v1" or target.get("kind") != "Deployment" or target.get("name") != deployment_meta.get("name"):
        raise PreflightError("HPA target must be the rendered apps/v1 Deployment")

    selector = spec.get("selector", {}).get("matchLabels", {})
    pod_labels = spec.get("template", {}).get("metadata", {}).get("labels", {})
    service_selector = service.get("selector", {})
    if not selector or any(pod_labels.get(k) != v for k, v in selector.items()):
        raise PreflightError("Deployment selector does not match its pod labels")
    if service_selector != selector:
        raise PreflightError("Service selector does not match Deployment selector")


def render(template: Path, profile_path: Path, values: dict[str, str]) -> list[dict[str, Any]]:
    profile = _profile(yaml.safe_load(profile_path.read_text(encoding="utf-8")))
    raw = template.read_text(encoding="utf-8")
    documents = [_replace(doc, values) for doc in yaml.safe_load_all(raw)]
    deployment = next((doc for doc in documents if doc and doc.get("kind") == "Deployment"), None)
    if deployment is None:
        raise PreflightError("template has no Deployment")
    deployment["spec"]["replicas"] = profile["replicas"]
    deployment["spec"]["strategy"] = {"type": profile["strategy"]}
    metadata = deployment["spec"]["template"].setdefault("metadata", {})
    metadata.setdefault("annotations", {})[SESSION_ANNOTATION] = profile["session_mode"]
    hpa = next((doc for doc in documents if doc and doc.get("kind") == "HorizontalPodAutoscaler"), None)
    if hpa is None:
        raise PreflightError("template has no HorizontalPodAutoscaler")
    hpa["spec"]["minReplicas"] = profile["hpa"]["minReplicas"]
    hpa["spec"]["maxReplicas"] = profile["hpa"]["maxReplicas"]
    return documents


def validate_serialized(path: Path, profile_path: Path) -> None:
    """Validate an already-rendered YAML file without contacting Kubernetes."""
    profile = _profile(yaml.safe_load(profile_path.read_text(encoding="utf-8")))
    raw = path.read_text(encoding="utf-8")
    if PLACEHOLDER.search(raw):
        raise PreflightError("rendered output contains unresolved placeholders")
    documents = list(yaml.safe_load_all(raw))
    if not documents or any(not isinstance(doc, dict) for doc in documents):
        raise PreflightError("rendered manifest must contain only non-empty YAML mappings")
    preflight(documents, profile)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--template", type=Path, default=Path("mcp-server/deploy/k8s-deployment.yaml"))
    parser.add_argument("--profile", type=Path, default=Path("mcp-server/deploy/profiles/process-local-single-instance.yaml"))
    parser.add_argument("--output", type=Path)
    parser.add_argument("--validate", type=Path, help="validate an existing rendered YAML file")
    parser.add_argument("--version", default="fixture")
    parser.add_argument("--build-time", default="1970-01-01T00:00:00Z")
    parser.add_argument("--git-commit", default="fixture")
    args = parser.parse_args(argv)
    try:
        if args.validate:
            validate_serialized(args.validate, args.profile)
            print(f"validation PASS: {args.validate}")
            return 0
        if not args.output:
            parser.error("--output is required unless --validate is supplied")
        values = {"VERSION": args.version, "BUILD_TIME": args.build_time, "GIT_COMMIT": args.git_commit}
        documents = render(args.template, args.profile, values)
        profile = yaml.safe_load(args.profile.read_text(encoding="utf-8"))
        preflight(documents, profile)
        output = yaml.safe_dump_all(documents, sort_keys=False, explicit_start=True)
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
