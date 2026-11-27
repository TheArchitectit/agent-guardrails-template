#!/usr/bin/env python3
"""Token-free, fixed client for the isolated Guardrails build worker.

This file is executed only from the public reusable workflow checkout.  It
never imports or executes the candidate checkout; the worker fetches the
candidate SHA independently and checks it out inside its disposable VM.
"""
from __future__ import annotations

import argparse
import base64
import hashlib
import http.client
import json
import os
import re
import ssl
import subprocess
import tempfile
import urllib.parse
import urllib.request
import time
from pathlib import Path

AUDIENCE = "https://guardrails-build-worker"
OIDC_SUFFIX = ".actions.githubusercontent.com"
MAX_RESPONSE = 65536


def canonical(value: object) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()


def oidc_token() -> str:
    raw_url = os.environ.get("ACTIONS_ID_TOKEN_REQUEST_URL", "")
    bearer = os.environ.get("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")
    parsed = urllib.parse.urlsplit(raw_url)
    if (parsed.scheme != "https" or not parsed.hostname
            or not parsed.hostname.endswith(OIDC_SUFFIX)
            or parsed.username or parsed.password or not bearer):
        raise RuntimeError("GitHub OIDC request environment is unavailable")
    query = [(k, v) for k, v in urllib.parse.parse_qsl(parsed.query)
             if k != "audience"]
    query.append(("audience", AUDIENCE))
    url = urllib.parse.urlunsplit(("https", parsed.netloc, parsed.path,
                                   urllib.parse.urlencode(query), ""))
    request = urllib.request.Request(url, headers={
        "Authorization": "Bearer " + bearer,
        "Accept": "application/json",
        "User-Agent": "guardrails-build-worker-client",
    })
    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, req, fp, code, msg, headers, newurl):
            return None
    with urllib.request.build_opener(NoRedirect()).open(request, timeout=5) as response:
        if response.geturl() != url:
            raise RuntimeError("OIDC endpoint redirected")
        data = json.loads(response.read(32769))
    token = data.get("value") if isinstance(data, dict) else None
    if not isinstance(token, str) or token.count(".") != 2 or len(token) > 16384:
        raise RuntimeError("invalid OIDC token response")
    return token


def claims(token: str) -> dict:
    try:
        part = token.split(".")[1]
        data = json.loads(base64.urlsafe_b64decode(part + "=" * (-len(part) % 4)))
    except Exception as exc:
        raise RuntimeError("OIDC claims cannot be decoded") from exc
    if not isinstance(data, dict):
        raise RuntimeError("OIDC claims are not an object")
    return data


def verify_receipt(receipt: dict, signature: str, key_path: str, token_claims: dict) -> None:
    try:
        sig = base64.b64decode(signature, validate=True)
    except Exception as exc:
        raise RuntimeError("invalid worker receipt signature encoding") from exc
    with tempfile.TemporaryDirectory(prefix="guardrails-receipt-") as directory:
        payload = Path(directory) / "receipt.json"
        sig_file = Path(directory) / "receipt.sig"
        payload.write_bytes(canonical(receipt))
        sig_file.write_bytes(sig)
        result = subprocess.run([
            "/usr/bin/openssl", "pkeyutl", "-verify", "-pubin", "-inkey",
            key_path, "-rawin", "-in", str(payload), "-sigfile", str(sig_file),
        ], stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL, env={"PATH": "/usr/bin:/bin"}, timeout=5)
        if result.returncode:
            raise RuntimeError("worker receipt signature rejected")
    for receipt_key, claim_key in (("repository", "repository"),
        ("repository_id", "repository_id"), ("event_name", "event_name"),
        ("ref", "ref"), ("run_id", "run_id"), ("run_attempt", "run_attempt"),
        ("check_run_id", "check_run_id"), ("candidate_sha", "sha")):
        if not token_claims.get(claim_key) or str(receipt.get(receipt_key)) != str(token_claims[claim_key]):
            raise RuntimeError("receipt identity is not bound to OIDC claims")
    for env_name, claim_name in (("GITHUB_REPOSITORY", "repository"),
        ("GITHUB_EVENT_NAME", "event_name"), ("GITHUB_SHA", "sha"),
        ("GITHUB_RUN_ID", "run_id"), ("GITHUB_RUN_ATTEMPT", "run_attempt")):
        expected = os.environ.get(env_name, "")
        if not expected or str(token_claims.get(claim_name, "")) != str(expected):
            raise RuntimeError("OIDC identity does not match current workflow environment")
    result = receipt.get("result")
    if (receipt.get("schema") != "guardrails-build-receipt/v1"
            or not isinstance(result, dict) or result.get("status") not in ("passed", "failed")):
        raise RuntimeError("worker receipt has an unsupported terminal result")
    # A signed failed receipt is terminal evidence and must be surfaced as a
    # failure, but it may legitimately have no image digest/protocol payload.
    if result.get("status") == "failed":
        return
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", str(result.get("image_digest", ""))):
        raise RuntimeError("worker passing receipt is not digest-bound")
    protocol = result.get("protocol", {})
    if (protocol.get("initialize") is not True or type(protocol.get("tools_list_count")) is not int
            or protocol["tools_list_count"] < 1
            or protocol.get("benign_tool") != "guardrail_advisor_list"
            or protocol.get("benign_result_count") != 9):
        raise RuntimeError("worker receipt lacks required protocol evidence")
    cleanup = result.get("cleanup", {})
    created, removed = cleanup.get("resources_created"), cleanup.get("resources_removed")
    prefix = f"guardrails-ci-{token_claims.get('run_id')}-{token_claims.get('run_attempt')}-"
    if not isinstance(created, list) or created != removed or not created or any(
            not isinstance(name, str) or not name.startswith(prefix) for name in created):
        raise RuntimeError("worker receipt lacks scoped cleanup evidence")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", required=True)
    parser.add_argument("--cert-sha256", required=True)
    parser.add_argument("--receipt-public-key", required=True)
    parser.add_argument("--ca-file", required=True)
    parser.add_argument("--receipt-file", required=True)
    args = parser.parse_args()
    parsed = urllib.parse.urlsplit(args.url)
    if (parsed.scheme != "https" or not parsed.hostname or parsed.username
            or parsed.password or parsed.path != "/v1/build" or parsed.query
            or parsed.fragment or parsed.port not in (None, 443, 8443)
            or not re.fullmatch(r"[0-9a-f]{64}", args.cert_sha256)):
        raise RuntimeError("worker URL or certificate pin is invalid")
    required = ("GITHUB_SHA", "GITHUB_EVENT_NAME", "GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT")
    if any(not os.environ.get(name) for name in required):
        raise RuntimeError("required workflow identity is missing")
    body = canonical({"candidate_sha": os.environ["GITHUB_SHA"].lower(),
                      "event_name": os.environ["GITHUB_EVENT_NAME"],
                      "run_id": os.environ["GITHUB_RUN_ID"],
                      "run_attempt": os.environ["GITHUB_RUN_ATTEMPT"]})
    context = ssl.create_default_context(cafile=args.ca_file)
    context.minimum_version = ssl.TLSVersion.TLSv1_2
    deadline = time.monotonic() + 70 * 60
    while (remaining := deadline - time.monotonic()) > 0:
        connection = http.client.HTTPSConnection(parsed.hostname, parsed.port or 443,
                                                context=context, timeout=min(remaining, 31 * 60))
        try:
            connection.connect()
            if hashlib.sha256(connection.sock.getpeercert(binary_form=True)).hexdigest() != args.cert_sha256:
                raise RuntimeError("worker TLS certificate pin mismatch")
            token = oidc_token()
            token_claims = claims(token)
            connection.request("POST", "/v1/build", body=body, headers={
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json",
                "Content-Length": str(len(body)), "Cache-Control": "no-store", "Connection": "close"})
            response = connection.getresponse()
            raw = response.read(MAX_RESPONSE + 1)
            if len(raw) > MAX_RESPONSE:
                raise RuntimeError("worker response exceeds size limit")
            if response.status == 200:
                data = json.loads(raw)
                break
            if response.status == 409:
                if json.loads(raw).get("status") not in ("running", "in_progress"):
                    raise RuntimeError("worker recorded an interrupted run; rerun the job")
            elif response.status not in (429, 503):
                raise RuntimeError(f"worker rejected request (HTTP {response.status})")
        except (http.client.RemoteDisconnected, TimeoutError, ConnectionResetError):
            # The request may have completed. The server's durable key prevents
            # a repeat of the VM build when this invocation retrieves its receipt.
            pass
        finally:
            connection.close()
        time.sleep(min(45, max(0, deadline - time.monotonic())))
    else:
        raise RuntimeError("worker admission/build exceeded 70-minute deadline")
    if not isinstance(data, dict) or not isinstance(data.get("receipt"), dict):
        raise RuntimeError("worker response has no receipt")
    verify_receipt(data["receipt"], data.get("signature_ed25519", ""),
                   args.receipt_public_key, token_claims)
    Path(args.receipt_file).write_text(json.dumps(data, sort_keys=True) + "\n")
    if data["receipt"]["result"]["status"] != "passed":
        raise RuntimeError("worker returned a signed terminal failure receipt")
    print(json.dumps(data, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
