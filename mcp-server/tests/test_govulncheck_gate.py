import json
import subprocess
import sys
from pathlib import Path

import pytest


SCRIPT = Path(__file__).parents[1] / "scripts" / "govulncheck_gate.py"
sys.path.insert(0, str(SCRIPT.parent))
from govulncheck_gate import GateError, classify


def message(key, value):
    return json.dumps({key: value})


CONFIG = {"protocol_version": "v1.0.0", "scanner_name": "govulncheck", "scanner_version": "v1.10.0", "scan_mode": "source", "scan_level": "symbol"}
SBOM = {"roots": ["example.test/project/cmd/server"]}


def finding(package=None, function=None, vuln_id="GO-2025-0001"):
    frame = {}
    if package is not None:
        frame["package"] = package
    if function is not None:
        frame["function"] = function
    return {"osv": vuln_id, "trace": [frame]}


def osv(vuln_id="GO-2025-0001"):
    return {"id": vuln_id}


def run_gate(tmp_path, stream):
    source = tmp_path / "scanner.json"
    report = tmp_path / "report.json"
    source.write_text(stream, encoding="utf-8")
    result = subprocess.run([sys.executable, str(SCRIPT), "--input", str(source), "--report", str(report)], capture_output=True, text=True)
    return result, report


@pytest.mark.parametrize(
    ("records", "exit_code", "category"),
    [
        ([message("osv", osv()), message("finding", finding("example.test/vuln", "DoThing"))], 1, "source-call-path"),
        ([message("osv", osv()), message("finding", finding("example.test/vuln"))], 0, "imported-package"),
        ([message("osv", osv()), message("finding", finding())], 0, "module-only"),
        ([], 0, None),
    ],
)
def test_classifies_findings_and_requires_package_evidence(tmp_path, records, exit_code, category):
    stream = "\n".join([message("config", CONFIG), message("SBOM", SBOM), *records]) + "\n"
    result, report_path = run_gate(tmp_path, stream)
    assert result.returncode == exit_code, result.stderr
    report = json.loads(report_path.read_text(encoding="utf-8"))
    if category:
        assert report["reachability_counts"][category] == 1
        assert report["findings"][0]["id"] == "GO-2025-0001"
        assert report["findings"][0]["package"] == ("example.test/vuln" if category != "module-only" else "")
    else:
        assert report["analyzed_root_count"] == 1
    assert "DoThing" not in report_path.read_text(encoding="utf-8")
    assert "DoThing" not in result.stdout
    if category:
        assert "GO-2025-0001" in result.stdout


@pytest.mark.parametrize(
    "stream",
    [
        "{bad json}\n",
        '{"config":{"protocol_version":"v1.0.0"},"config":{"protocol_version":"v1.0.0"}}\n',
        message("config", CONFIG),  # No SBOM roots, so no proof analysis ran.
        "\n".join([message("config", CONFIG), message("SBOM", {"roots": []})]),
        "\n".join([message("config", CONFIG), message("SBOM", SBOM), message("finding", {"osv": "GO-1", "trace": []})]),
        "\n".join([message("config", CONFIG), message("SBOM", SBOM), message("finding", finding(vuln_id="GO-UNKNOWN"))]),
    ],
)
def test_fails_closed_on_invalid_or_empty_output(tmp_path, stream):
    result, report_path = run_gate(tmp_path, stream)
    assert result.returncode == 2
    assert "incomplete or invalid scan" in result.stderr
    assert not report_path.exists()


def test_scanner_failure_is_preserved_by_caller():
    with pytest.raises(GateError, match="scanner exited with status 7"):
        classify("", scanner_exit_code=7)


def test_accepts_whitespace_separated_pretty_json_records(tmp_path):
    stream = "\n".join([json.dumps({"config": CONFIG}, indent=2), json.dumps({"SBOM": SBOM}, indent=2)]) + "\n"
    result, report_path = run_gate(tmp_path, stream)
    assert result.returncode == 0, result.stderr
    assert json.loads(report_path.read_text(encoding="utf-8"))["analyzed_root_count"] == 1


def test_allows_unreferenced_osv_records(tmp_path):
    stream = "\n".join([message("config", CONFIG), message("SBOM", SBOM), message("osv", osv("GO-UNREFERENCED"))])
    result, report_path = run_gate(tmp_path, stream)
    assert result.returncode == 0, result.stderr
    assert json.loads(report_path.read_text(encoding="utf-8"))["reachability_counts"]["module-only"] == 0
