# Tasks

## 1. Team manager timestamps

- [x] 1.1 Replace naive UTC creation and arithmetic in `scripts/team_manager.py` with aware UTC values, then run `python -m pytest scripts/test_team_manager.py` with coverage for the existing JSON timestamp keys and terminal `Z` format.

## 2. Log aggregation timestamps

- [x] 2.1 Use aware UTC boundaries and normalize parsed timestamps in `scripts/log_aggregator.py`, add focused relative-window and `Z`/offset parsing tests, then run `python -m pytest scripts/test_log_aggregator.py`.

## 3. Python 3.12 compatibility

- [x] 3.1 Run the Python script suite on Python 3.12 and verify the migrated timestamp paths emit no naive-UTC deprecation warnings while preserving timestamp keys and serialized values.
