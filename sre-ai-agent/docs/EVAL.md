# Evaluation

## Quick Start

```bash
make eval
```

This runs all eval cases and writes a report to `tests/eval/report.json`.

## Test Data

- **Codebase:** `tests/data/code/sample-app/` — a Go app with 3 seeded bugs (nil pointer, SQL injection, missing HTTP timeout)
- **Logs:** `tests/data/logs/` — log fixtures for each eval case

## Adding a Case

Add an entry to `tests/eval/cases.json`:

```json
{
  "id": "case_004",
  "name": "nil pointer in fetchUser",
  "logs_path": "tests/data/logs/case_004.log",
  "codebase_path": "tests/data/code/sample-app",
  "ground_truth": {
    "function": "main.fetchUser",
    "fix_summary": "Add nil check on user pointer before dereference"
  }
}
```

## Interpreting Results

The runner classifies each case as:

| Result | Condition |
|---|---|
| `PASS` | Top hypothesis targets the ground-truth function |
| `FAIL` | No hypothesis matches, or a different function is ranked higher |

The report includes accuracy (% of cases with `PASS`), per-case details, and duration.

## Metrics Tracked

- **Accuracy** — % of cases where the top hypothesis matches `ground_truth.function`
- **Latency** — per-case and total analysis time
- **Coverage** — which seeded bugs are detected
