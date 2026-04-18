# Eval Fixtures

Each fixture file is a frozen golden test set linking a traitex snapshot to a hand-curated list of expected tasks.

## Format

```json
{
  "version": 1,
  "snapshot_id": "<UUID of a traitex processing snapshot>",
  "user_id": "<UUID of the user the events belong to>",
  "expected_tasks": [
    {
      "title": "Review Q4 budget proposal",
      "description": "Read the attached spreadsheet and comment by Friday.",
      "duration_minutes": 30,
      "priority": 7,
      "deadline": "2026-04-17T23:59:59Z",
      "start_time": null,
      "category": "work"
    }
  ]
}
```

## Curating a new fixture

1. Pick a traitex snapshot you want to evaluate against (or create one with `CreateProcessingSnapshot`).
2. Run the eval script once with an empty `expected_tasks: []` list to see what tasker currently generates.
3. Copy the `unmatched_generated` entries from the report into `expected_tasks`.
4. **Edit each task to reflect what _should_ have been generated** — base this on the source events, not on tasker's draft output. Drop spurious tasks, add missing ones.
5. Save the file as `fixtures/golden_v<N>.json`.
6. Re-run the script: F1 now measures how close tasker comes to your curated set.

## Files

- `golden_v1.json` — first golden set (create this after first real run).
