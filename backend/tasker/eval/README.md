# Tasker F1 Evaluation

Offline script that scores tasker's end-to-end task generation (clustering + LLM) against a curated set of expected tasks for a fixed traitex snapshot.

**Output:** precision, recall, F1 at task level, plus per-field accuracy on matched pairs.

---

## Prerequisites

1. A traitex snapshot exists in the traitex DB (use `CreateProcessingSnapshot` or `CreateSnapshotFromCurrentMoment`).
2. An eval SQS queue exists (Yandex MQ). Export its URL:
   ```
   export EVAL_QUEUE_URL=https://message-queue.api.cloud.yandex.net/.../tasker-eval-events.fifo
   ```
3. `secrets.env` is populated (run `make secrets` from the repo root).
4. A fixture JSON exists at `backend/tasker/eval/fixtures/<name>.json` (see [Curating a fixture](#curating-a-fixture)).

---

## Running the eval

```bash
# Start the eval tasker (uses tasker_eval DB + eval queue)
cd backend
make tasker-eval/run

# Run the eval script
go run ./tasker/eval/cmd/f1 \
  --fixture=tasker/eval/fixtures/test1.json \
  --tasker-http=http://localhost:9091 \
  --eval-queue-url=https://message-queue.api.cloud.yandex.net/b1gfkhf234165rrluh60/dj60000000kjtgs303cs/tasker-eval-events.fifo \
  --report=./tasker/eval/fixtures/report.json --report-only=true
```

`docker-compose.eval.yml` exposes two tasker ports locally: `8091` for gRPC and `9091` for HTTP. The eval script uses the HTTP test endpoint, so `--tasker-http` must point to `http://localhost:9091`.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--fixture` | _(required)_ | Path to fixture JSON |
| `--traitex-grpc` | _(required)_ | `host:port` of traitex gRPC server |
| `--tasker-http` | `http://localhost:9091` | Base URL of eval tasker HTTP server |
| `--eval-queue-url` | _(required)_ | Eval SQS/YMQ queue URL |
| `--report` | _(stdout only)_ | Write report JSON to this path |
| `--runs` | `1` | Number of independent runs to average |
| `--poll-interval` | `10s` | How often to log and refresh the current generated task count |
| `--overall-timeout` | `15m` | How long to wait before collecting the latest task list (minimum `15m`) |

The script now logs snapshot replay and task wait progress to stderr so you can see how many tasks have appeared while it waits.

---

## Curating a fixture

1. **Bootstrap:** write a minimal fixture with the snapshot ID, user ID, and `"expected_tasks": []`.
2. **First run:** the script reports all `unmatched_generated` tasks.
3. **Edit:** copy those tasks into `expected_tasks`, correcting each one to match what *should* have been generated (based on the raw events, not tasker's output).
4. **Save** as `fixtures/golden_v<N>.json`.
5. **Re-run** — F1 now measures quality against your curated set.

See `fixtures/README.md` for the full fixture format.

---

## Interpreting the report

```json
{
  "precision": 0.75,   // of tasks generated, what fraction were correct
  "recall":    0.67,   // of expected tasks, what fraction were found
  "f1":        0.71,   // headline metric
  "matched_field_accuracy": {
    "description":      {"passed": 5, "total": 9},
    "duration_minutes": {"passed": 7, "total": 9},
    ...
  }
}
```

- **F1 < 0.5** → the pipeline is producing many wrong or missing tasks. Check `unmatched_generated` and `unmatched_expected`.
- **F1 ≥ 0.8** → good task generation quality.
- Secondary fields show which attributes (duration, priority, category, deadline) the LLM gets right even on matched tasks.
