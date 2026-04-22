# F1 Raise Plan

Current baseline from `backend/tasker/eval/fixtures/report.json`:

- Precision: `0.4375`
- Recall: `0.4667`
- F1: `0.4516`

Main observed failure modes:

- Too many non-actionable clusters become tasks, which hurts precision.
- Related events split across multiple clusters, which hurts recall.
- Similar-but-different work items merge into one cluster, which hurts both precision and recall.
- One cluster is forced into exactly one task, which loses multi-action cases and creates over-broad tasks.
- Generation is one-shot and weakly grounded, so titles often lose key identifiers.

## Priority Order

### 1. Add actionable gating and start with `no_task | one_task`

Goal:
Reduce false positives from FYI, spam, receipts, promos, and weak-signal clusters without changing the current assumption that one actionable cluster produces at most one task.

Concrete tasks:

1. Change task generation contract from `GenerateTask(events) -> one task` to `GenerateTask(cluster) -> no_task | one_task`.
2. Add a cluster-level actionability classifier step before task extraction.
3. Require each generated task to include evidence references to specific event IDs.
4. Keep the current `1 actionable cluster -> 1 task` restriction for the first iteration.
5. Keep clusterization as a separate step, but let generation return:
   - `no_task`
   - `one_task`
6. Add a follow-up milestone for future `many_tasks` support, but do not include it in the first implementation.
7. Extend eval reports to show:
   - clusters skipped as non-actionable
   - taskless cluster rate
   - generated task count per cluster

Why first:

- This is the fastest way to raise precision.
- It isolates the precision problem first and keeps the rollout easier to evaluate.

Expected F1 impact:

- High precision gain
- Low to medium recall gain

Dependencies:

- None

### 2. Replace pure nearest-centroid assignment with hybrid entity-aware clustering

Goal:
Reduce wrong merges between semantically similar but distinct work items and improve grouping of related events that share explicit identifiers.

Concrete tasks:

1. Keep embedding similarity, but add symbolic features to cluster scoring:
   - PR number
   - repository name
   - workflow name
   - CVE ID
   - domain name
   - sender or thread identity
   - source type
   - time distance
2. Build a candidate scoring function that combines embedding similarity with feature overlap instead of selecting by centroid similarity alone.
3. Introduce hard negative rules for obviously incompatible candidates, for example:
   - different PR numbers
   - different repositories for PR-review tasks
   - different domains for renewal tasks
4. Make clustering thresholds configurable instead of hardcoded.
5. Log cluster assignment decisions with per-feature score breakdown for offline analysis.
6. Add offline replay tooling to compare old vs new assignment decisions on eval fixtures.

Why second:

- Current bad merges are a major source of both false positives and false negatives.
- This improves the quality of the input to generation instead of only trying to fix mistakes later.

Expected F1 impact:

- High precision gain
- High recall gain

Dependencies:

- None, but pairs well with task 3

### 3. Support multi-membership event-to-cluster links when evidence is strong

Goal:
Preserve the separate clusterization step while allowing one event to contribute to multiple legitimate tasks.

Concrete tasks:

1. Replace single-valued `events.cluster_id` ownership with an `event_cluster_memberships` relation.
2. Keep one primary cluster if needed for backward compatibility in reads, but stop treating it as the only valid association.
3. Allow membership creation only when explicit evidence justifies it, such as a single email mentioning:
   - several PRs
   - several repos
   - several domains
   - several alerts that should roll up separately
4. Add membership confidence and membership reason fields.
5. Update closure logic so cluster generation reads events through memberships.
6. Prevent explosion by limiting memberships per event and requiring a stronger threshold for secondary memberships.

Why third:

- This directly addresses the user constraint that clusterization stays separate.
- It raises recall in cases where one event legitimately belongs to more than one task stream.

Expected F1 impact:

- Medium precision gain
- High recall gain

Dependencies:

- Task 2 recommended first

### 4. Add a pre-generation cluster refinement pass: split, merge, reassign

Goal:
Repair bad clusters before any LLM call.

Concrete tasks:

1. Add a refinement step that runs on closable clusters before generation.
2. Split clusters that contain multiple incompatible entity groups or multiple distinct action threads.
3. Merge sibling clusters when they share a strong identifier set and have compatible timing.
4. Reassign outlier events whose similarity to the chosen cluster is much lower than to another nearby cluster.
5. Store refinement decisions for auditability.
6. Add eval diagnostics for:
   - clusters split
   - clusters merged
   - events reassigned

Why fourth:

- The current pipeline has no recovery once an early clustering mistake is made.
- This lets us keep online clustering simple while correcting mistakes at closure time.

Expected F1 impact:

- Medium precision gain
- High recall gain

Dependencies:

- Task 2 strongly recommended first
- Task 3 helpful

### 5. Move to two-stage grounded generation with verification

Goal:
Improve title correctness, identifier preservation, and resistance to hallucinated tasks.

Concrete tasks:

1. Stage 1 extractor produces candidate tasks with:
   - title
   - description
   - evidence event IDs
   - extracted entities
   - confidence
2. Stage 2 verifier checks each candidate for:
   - evidence support in cited events
   - preserved identifiers in title
   - duplication against other candidates
   - whether the task is actionable vs FYI
3. Reject or down-rank candidates with weak support.
4. Tighten the prompt to require copying critical identifiers exactly when present.
5. Add structured failure logging for rejected candidates.
6. Evaluate title-only quality separately from field quality to make regressions visible.

Why fifth:

- Better grounding improves precision, but it helps most after cluster quality improves.
- Otherwise the verifier still receives mixed or fragmented evidence.

Expected F1 impact:

- High precision gain
- Medium recall gain

Dependencies:

- Task 1 recommended first
- Benefits from tasks 2 and 4

### 6. Later phase: allow `many_tasks` for a single cluster

Goal:
Recover recall for genuinely multi-action clusters, but only after the simpler `no_task | one_task` flow is stable.

Concrete tasks:

1. Change generation from `no_task | one_task` to `no_task | one_task | many_tasks`.
2. Update DB schema to remove the effective `1 cluster -> 1 task` restriction.
3. Add intra-cluster task dedup and ranking.
4. Require each generated task to cite supporting event IDs.
5. Add eval slices specifically for multi-action clusters to verify recall actually improves.

Why later:

- This is mainly a recall optimization, but it introduces more room for false positives.
- It is easier to judge once cluster quality and actionability gating are already improved.

Expected F1 impact:

- Medium precision risk
- Medium to high recall gain

Dependencies:

- Task 1 must be complete first
- Tasks 2 and 4 strongly recommended first

## Cross-Cutting Work

These should be done alongside the priority items above.

### A. Improve observability for F1 debugging

1. Persist per-task evidence event IDs in generated output.
2. Persist cluster assignment explanations for each event.
3. Add per-cluster eval dumps showing:
   - source events
   - chosen cluster scores
   - refinement actions
   - generated tasks
4. Add confusion summaries for top false-positive and false-negative patterns.

### B. Make experiments cheap to run

1. Expose clustering thresholds and scoring weights via config.
2. Add offline replay mode for clusterization without reprocessing everything live.
3. Add side-by-side eval runner support for comparing two strategies on the same fixture.
4. Save reports with strategy metadata so results are reproducible.

### C. Expand eval coverage

1. Add more fixtures with different failure modes:
   - PR review traffic
   - security alerts
   - billing and banking mail
   - marketing/promotional noise
   - course and signup flows
2. Track precision and recall by task family, not just globally.
3. Add a small hand-labeled cluster-quality benchmark in addition to end-to-end task F1.

## Recommended rollout

1. Task 1: actionable gating + `no_task | one_task`
2. Task 2: hybrid entity-aware clustering
3. Task 4: split/merge/reassign refinement pass
4. Task 3: multi-membership event-to-cluster links
5. Task 5: two-stage grounded generation with verification
6. Task 6: later `many_tasks` support per cluster

## Success criteria

Short-term target:

- Raise F1 from `0.45` to `0.60+`

Medium-term target:

- Reach `0.75+` on the current fixture set

Guardrails:

- Do not regress precision while chasing recall.
- Keep clusterization as a distinct pipeline stage.
- Every generated task should be traceable back to concrete source events.
