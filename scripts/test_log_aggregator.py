"""Timezone-aware parsing and filtering tests for the log aggregator."""

import json
from datetime import datetime, timedelta, timezone
from pathlib import Path

from scripts.log_aggregator import LogAggregator, parse_time_expr


def test_relative_time_expression_returns_aware_utc() -> None:
    result = parse_time_expr("30m")

    assert result.tzinfo is timezone.utc
    assert result < datetime.now(timezone.utc)
    assert datetime.now(timezone.utc) - result <= timedelta(minutes=31)


def test_absolute_z_offset_and_naive_times_normalize_to_utc() -> None:
    assert parse_time_expr("2026-11-25T10:00:00Z") == datetime(
        2026, 11, 25, 10, tzinfo=timezone.utc
    )
    assert parse_time_expr("2026-11-25T12:00:00+02:00") == datetime(
        2026, 11, 25, 10, tzinfo=timezone.utc
    )
    assert parse_time_expr("2026-11-25T10:00:00") == datetime(
        2026, 11, 25, 10, tzinfo=timezone.utc
    )


def test_timestamp_parser_normalizes_legacy_and_offset_records(tmp_path: Path) -> None:
    aggregator = LogAggregator("fixture", tmp_path)

    for raw in (
        "2026-11-25T10:00:00Z",
        "2026-11-25T12:00:00+02:00",
        "2026-11-25T10:00:00",
        "2026-11-25T10:00:00.123456",
    ):
        parsed = aggregator._parse_timestamp(raw)
        assert parsed == datetime(2026, 11, 25, 10, 0, 0, parsed.microsecond, timezone.utc)
        assert parsed.tzinfo is timezone.utc


def test_query_compares_z_offset_and_legacy_records_with_aware_window(tmp_path: Path) -> None:
    log_dir = tmp_path / "logs"
    log_dir.mkdir()
    records = [
        {"timestamp": "2026-11-25T10:30:00Z", "event": "z"},
        {"timestamp": "2026-11-25T09:45:00-01:00", "event": "offset"},
        {"timestamp": "2026-11-25T09:45:00-02:00", "event": "latest_offset"},
        {"timestamp": "2026-11-25T10:15:00", "event": "legacy"},
        {"timestamp": "2026-11-25T11:10:00Z", "event": "outside"},
        {"timestamp": None, "event": "missing"},
        {"timestamp": 123, "event": "numeric"},
        {"timestamp": "not a timestamp", "event": "unparseable"},
    ]
    (log_dir / "fixture.jsonl").write_text(
        "".join(json.dumps(record) + "\n" for record in records), encoding="utf-8"
    )
    aggregator = LogAggregator("fixture", tmp_path)

    result = aggregator.query(
        since=datetime(2026, 11, 25, 10, tzinfo=timezone.utc),
        until=datetime(2026, 11, 25, 11, tzinfo=timezone.utc),
        sources=["structured"],
    )

    assert [entry["event"] for entry in result] == ["legacy", "z", "offset"]

    summary = aggregator.get_summary()
    assert summary["structured_logs"]["entries"] == len(records)
    assert summary["time_range"]["oldest"] == "2026-11-25T10:15:00"
    assert summary["time_range"]["newest"] == "2026-11-25T09:45:00-02:00"

    all_entries = aggregator.query(sources=["structured"])
    assert [entry["event"] for entry in all_entries] == [
        "legacy", "z", "offset", "outside", "latest_offset",
        "missing", "numeric", "unparseable",
    ]
