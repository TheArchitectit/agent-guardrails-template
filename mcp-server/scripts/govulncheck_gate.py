#!/usr/bin/env python3
"""Classify govulncheck v1.10.0 JSON without publishing source traces."""

from __future__ import annotations

import argparse
import json
import sys
from collections import Counter
from pathlib import Path
from typing import Any


PROTOCOL_VERSION = "v1.0.0"
REACHABILITY = ("source-call-path", "imported-package", "module-only")


class GateError(ValueError):
    pass


def _reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise GateError(f"duplicate JSON field: {key}")
        result[key] = value
    return result


def _one_key_object(obj: Any, label: str) -> tuple[str, Any]:
    if not isinstance(obj, dict):
        raise GateError(f"{label} must be an object")
    present = [(key, value) for key, value in obj.items() if value is not None]
    if len(present) != 1:
        raise GateError(f"each stream message must contain exactly one payload: {label}")
    return present[0]


def classify(text: str, scanner_exit_code: int = 0) -> dict[str, Any]:
    if scanner_exit_code != 0:
        raise GateError(f"scanner exited with status {scanner_exit_code}")
    messages: list[tuple[str, Any]] = []
    decoder = json.JSONDecoder(object_pairs_hook=_reject_duplicate_keys)
    offset = 0
    while offset < len(text):
        while offset < len(text) and text[offset].isspace():
            offset += 1
        if offset == len(text):
            break
        try:
            obj, end = decoder.raw_decode(text, offset)
        except json.JSONDecodeError as exc:
            line_no = text.count("\n", 0, offset) + 1
            raise GateError(f"invalid JSON at line {line_no}") from exc
        messages.append(_one_key_object(obj, f"record {len(messages) + 1}"))
        offset = end
        if offset < len(text) and not text[offset].isspace():
            raise GateError("JSON records must be separated by whitespace")
    if not messages or messages[0][0] != "config":
        raise GateError("missing initial config message")
    config = messages[0][1]
    if not isinstance(config, dict) or config.get("protocol_version") != PROTOCOL_VERSION:
        raise GateError("unsupported or missing JSON protocol version")
    if config.get("scan_mode") != "source" or config.get("scan_level") != "symbol":
        raise GateError("expected a source-mode symbol scan")

    roots: list[str] = []
    findings: list[dict[str, str]] = []
    osv_ids: set[str] = set()
    for kind, payload in messages[1:]:
        if kind == "config":
            raise GateError("duplicate config message")
        if kind == "SBOM":
            if not isinstance(payload, dict):
                raise GateError("SBOM payload must be an object")
            candidate = payload.get("roots")
            if candidate is not None:
                if not isinstance(candidate, list) or any(not isinstance(x, str) or not x.strip() for x in candidate):
                    raise GateError("invalid SBOM roots")
                roots.extend(candidate)
        elif kind == "osv":
            if not isinstance(payload, dict) or not isinstance(payload.get("id"), str) or not payload["id"].strip():
                raise GateError("invalid OSV record")
            osv_ids.add(payload["id"])
        elif kind == "finding":
            if not isinstance(payload, dict):
                raise GateError("finding payload must be an object")
            vuln_id = payload.get("osv")
            trace = payload.get("trace")
            if not isinstance(vuln_id, str) or not vuln_id.strip() or not isinstance(trace, list) or not trace:
                raise GateError("finding is missing advisory id or trace")
            first = trace[0]
            if not isinstance(first, dict):
                raise GateError("finding trace frame must be an object")
            package, function = first.get("package"), first.get("function")
            if package is not None and not isinstance(package, str):
                raise GateError("finding trace package must be a string")
            if function is not None and not isinstance(function, str):
                raise GateError("finding trace function must be a string")
            if function:
                category = "source-call-path"
            elif package:
                category = "imported-package"
            else:
                category = "module-only"
            findings.append({"id": vuln_id, "reachability": category, "package": package or ""})
        elif kind == "progress":
            if not isinstance(payload, dict):
                raise GateError("progress payload must be an object")
            # Progress is informational in the govulncheck protocol and is not
            # evidence that source packages were successfully analyzed.
        else:
            raise GateError(f"unexpected stream message type: {kind}")

    unique_roots = sorted(set(roots))
    if not unique_roots:
        raise GateError("scan supplied no analyzed package roots")
    dangling = sorted({finding["id"] for finding in findings} - osv_ids)
    if dangling:
        raise GateError("finding references missing OSV record(s): " + ", ".join(dangling))
    counts = Counter(f["reachability"] for f in findings)
    return {
        "scanner": config.get("scanner_name", "govulncheck"),
        "scanner_version": config.get("scanner_version", "unknown"),
        "protocol_version": PROTOCOL_VERSION,
        "analyzed_root_count": len(unique_roots),
        "reachability_counts": {category: counts[category] for category in REACHABILITY},
        "findings": sorted(findings, key=lambda item: (item["reachability"], item["id"], item["package"])),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", required=True, type=Path, help="govulncheck -json stream")
    parser.add_argument("--report", required=True, type=Path, help="redacted JSON summary output")
    args = parser.parse_args()
    try:
        report = classify(args.input.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, GateError) as exc:
        print(f"govulncheck gate: incomplete or invalid scan: {exc}", file=sys.stderr)
        return 2
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    counts = report["reachability_counts"]
    print("govulncheck reachability summary (source paths and traces omitted):")
    for category in REACHABILITY:
        print(f"  {category}: {counts[category]}")
    for finding in report["findings"]:
        package = finding["package"] or "(module only)"
        print(f"  {finding['reachability']}: {finding['id']} [{package}]")
    if counts["source-call-path"]:
        print("govulncheck gate: source-call-path findings fail this check", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
