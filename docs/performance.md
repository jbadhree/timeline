# Performance and concurrency

Requirement ID: TL-PERF. Status: required engineering policy; implementation and measurements pending. Timeline must be efficient and safe under concurrent use. No latency or throughput claim has been measured yet.

## Bounded parallel work

Parallelize independent discovery, parsing, hashing, and validation through a bounded worker pool. Bound queued items, open files, subprocesses, and memory as well as goroutines. Stream large scans instead of retaining every document body. Start with a conservative worker default, then measure on representative filesystems. [Go errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) offers cancellation and concurrency limits; a worker pool using standard primitives is also acceptable.

Pass `context.Context` from command entry through core and adapters, SQL operations, and Git subprocesses. Stop scheduling on cancellation and join workers before returning. Validation diagnostics are data to aggregate; reserve fail-fast cancellation for operational failures. Sort diagnostics and results independently of worker completion order.

Parallel validation does not imply parallel registration. Use one authoritative acceptance writer for the relevant Git/root state across processes. An in-process mutex is insufficient. Do expensive preparation outside the critical section, then lock, re-read the expected accepted revision, selected file hashes, configuration/type definitions, and relevant validation dependencies such as duplicate-ID discovery, reject stale work, and validate registration timing at the actual publication protocol boundary. Define one lock order and bounded wait/cancellation behavior. Recovery must reconcile partial effects before another writer proceeds.

Before implementing writes, specify the publication point, durable intent/record layout, external-edit handling, and crash recovery states in the architecture. This guide does not claim a file save, Git ref update, and database transaction can form one atomic transaction. Consider expected-old-value Git ref updates and an isolated Git index to protect unrelated staged changes. Keep durable records reachable through normal full clones. [Git update-ref](https://git-scm.com/docs/git-update-ref) supports conditional ref updates; cross-resource recovery is our responsibility.

## Initial SQLite backend

These settings apply to the initial SQLite adapter, not to every future backend. Each supported backend must meet [TL-DATABASE](requirements/TL-DATABASE.md) and document its own measured connection/concurrency policy.

Start with one writer connection and short batched transactions. A small reader pool and WAL are candidates to benchmark, not prerequisites for high performance. WAL allows concurrent readers and one writer, still permits busy errors, and long readers can prevent checkpoint completion and allow the WAL to grow. It requires local shared-memory coordination and does not work over network filesystems. Use bounded busy handling; do not turn off durability simply to improve results. See [SQLite WAL](https://sqlite.org/wal.html).

Before selecting a driver, inspect its bundled runtime SQLite version and upstream fixes. The WAL documentation currently identifies the WAL-reset corruption fix in SQLite 3.51.3 and backports 3.44.6/3.50.7; verify the selected version includes that fix rather than assuming a recent Go module tag is sufficient.

Configure required connection-local pragmas on every connection using the driver's supported mechanism and verify them. Use the transaction handle consistently; do not hold the only connection and request another through `sql.DB`. Close result sets promptly. Avoid Git calls, filesystem scans, or external validation inside a SQL transaction. Inspect pool wait metrics when diagnosing contention. See [Go connection management](https://go.dev/doc/database/manage-connections).

Use parameterized queries, appropriate indexes for measured filters, query-plan inspection, and bounded result processing. Avoid a Git subprocess per item; batch object/history reads where supported. Do not use file timestamps alone as proof that content is unchanged. Every optimization must preserve detection of relevant pending edits.

## Filesystems and concurrent processes

Identify Git paths through Git, including linked worktrees; `.git` need not be a directory. Document whether runtime data roots in linked worktrees are supported and how shared refs are locked before enabling that configuration. Development worktree isolation alone does not settle application locking.

Define replacement and durability behavior per supported OS. Same-directory temporary files and rename are useful, but [Go's rename contract](https://pkg.go.dev/os#Rename) does not promise atomicity on non-Unix systems. Hash rechecks do not make arbitrary external editors cooperate: document the remaining race and never report bytes as validated if they differ from the validated snapshot. Recovery must not overwrite later external edits.

## Benchmarks and acceptance evidence

Proposed workload scales are 100, 10,000, and 100,000 items, with varied body sizes, history depth, metadata cardinality, pending-change fractions, and worker limits. These are evaluation targets, not supported-size promises. Benchmark cold rebuild, warm unchanged check, single-item acceptance, large selected set, filtered list, and long history separately.

Record elapsed latency distributions, allocations, peak memory, Git subprocess counts, lock wait, SQLite busy/pool wait, and total work. Include machine, OS, filesystem, CPU limits, storage, Go/Git/SQLite versions, dataset seed, command, and warm/cold cache conditions. Avoid claiming a cleared OS cache unless the runner actually controls it.

Use Go benchmarks with repeated runs and allocation reporting; compare with [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat). Profile before optimizing: CPU/heap for expensive work and mutex/block profiles or traces for contention. Collect instrumentation separately when it distorts measurements. See [Go diagnostics](https://go.dev/doc/diagnostics).

Before the first feature release, record an actual baseline and explicit reviewed latency/memory/resource budgets for representative workloads. Require explanations and measured evidence for regressions. Run stable comparisons on controlled runners; do not encode noisy wall-time thresholds as flaky unit tests. Stress separate CLI processes and test cancellation, failure, and restart alongside throughput.

Acceptance criteria:

- TL-PERF/AC-1: Resource limits and cancellation behavior are documented and verified under load.
- TL-PERF/AC-2: Concurrent executable tests verify that documented scenarios produce no lost accepted updates, duplicate registration, or silent stale acceptance.
- TL-PERF/AC-3: Reproducible fixtures and environment-tagged measurements exist for the workloads above.
- TL-PERF/AC-4: Measured budgets are recorded before a performance claim or feature release; correctness suites still pass after optimizations.
