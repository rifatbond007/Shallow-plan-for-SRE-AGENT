# SRE AI Agent - Implementation Plan (v2)

> **Source of truth:** [`SPECIFICATION.md`](./SPECIFICATION.md), §7.
> This document is the week-by-week breakdown.
> If anything here contradicts the spec, the spec wins.

---

## 0. What changed from v1

The previous 10-week flat plan tried to ship Terraform, AWS-grade infra,
ELK, vector DB, chaos engineering, etc. **That's not a thesis plan; that's
a startup roadmap.** This v2 plan replaces it with three realistic phases
and a "demo gate" at the end of each.

| v1 (old) | v2 (new) |
|----------|----------|
| 10 weeks, flat feature list | 3 phases (8 weeks), each with a demo gate |
| AWS-first, Terraform included | Local-first (Minikube/Kind), no Terraform |
| Vector DB embedded | No vector DB |
| 5 features, no clear MVP | MVP first, fix later |

---

## 1. Phase 0 — Foundation (Week 1)

**Goal:** build and run a "hello world" HTTP server in Docker.

### Tasks
- [ ] `go mod init github.com/rifatbond007/sre-ai-agent`
- [ ] Add core deps: gin, zap, prometheus/client_golang, anthropic-sdk-go, env loader, websocket
- [ ] `internal/config` with all env vars from SPECIFICATION.md §4.1
- [ ] `pkg/logger` (zap), `pkg/metrics` (prometheus collectors defined in §4.7)
- [ ] `cmd/agent/main.go`: load config → build logger → start gin → register `/healthz` and `/readyz`
- [ ] `Dockerfile` multi-stage build, non-root, `/healthz` healthcheck
- [ ] `docker-compose.yml` with `sre-agent` service
- [ ] `.env.example` with every env var
- [ ] `Makefile` targets: `build`, `test`, `run`, `docker-build`, `docker-run`, `lint`
- [ ] First GitHub Actions workflow: lint + test on push

### Deliverable
- `docker compose up` → curl `:8080/api/v1/healthz` returns 200.

---

## 2. Phase 1 — MVP: detect + hypothesize (Weeks 2–4)

**Demo gate:** *give it a log, get a ranked hypothesis with a code ref.*

### Week 2 — Ingestion
- [ ] `internal/ingest/nginx_access.go` (combined log format regex)
- [ ] `internal/ingest/nginx_error.go` (nginx error format)
- [ ] `internal/ingest/app_json.go` (JSON, including stack traces)
- [ ] `internal/ingest/normalizer.go` (`LogEntry`, severity mapping)
- [ ] Fixtures in `tests/data/logs/*.log` + golden files `*.expected.json`
- [ ] Unit tests covering all three formats and at least one malformed-line case each

### Week 3 — Codebase + grouping
- [ ] `internal/ingest/incident.go` — sliding-window grouper
- [ ] `internal/codebase/scanner.go` — `filepath.WalkDir`, ignore vendor/tests
- [ ] `internal/codebase/ast.go` — `go/parser` + `go/ast`, build `Function` records
- [ ] `internal/codebase/callgraph.go` — intra-package call edges
- [ ] `internal/codebase/linker.go` — score functions vs an `Incident`
- [ ] AST cache: serialize index to `CodebaseCacheDir/index.json`
- [ ] `tests/data/code/sample-app/` — small Go HTTP service with 3 seeded bugs:
  - Bug A: nil-pointer dereference in handler
  - Bug B: missing HTTP client timeout
  - Bug C: DB error path returning 500

### Week 4 — Analysis engine + first endpoint
- [ ] `internal/analysis/types.go` — `Hypothesis`, `Fix`, `AnalysisResult`
- [ ] `internal/analysis/patterns.go` — built-in signature library (7 entries from SPEC §4.4)
- [ ] `internal/analysis/context.go` — prompt builder with token budget
- [ ] `internal/analysis/claude.go` — Anthropic SDK wrapper with retries
- [ ] `internal/analysis/ranker.go` — combine pattern + LLM confidence
- [ ] `internal/analysis/engine.go` — orchestrator (sync + stream variants)
- [ ] `internal/api/handlers_analyze.go` — `POST /api/v1/analyze`
- [ ] `internal/api/handlers_get.go` — `GET /api/v1/hypotheses/:id`
- [ ] `internal/storage/cache.go` — LRU + TTL
- [ ] `prompts/system.txt`, `prompts/hypothesis.txt` (see SPEC §5)
- [ ] Eval harness v1: 10 labeled cases, top-1 accuracy reported

### Deliverable
- `POST /api/v1/analyze {logs, codebase_path}` returns a ranked list of
  hypotheses with `suspect_code` for at least 3 seeded bugs.
- Top-1 accuracy ≥ 0.7 against the 10-case eval set.

---

## 3. Phase 2 — V1: fix suggestion + streaming (Weeks 5–6)

**Demo gate:** *give it a log, get hypothesis AND a proposed code patch.*

### Week 5 — Fix generation
- [ ] `prompts/fix.txt` (see SPEC §5.3)
- [ ] Extend `claude.go` to make a second call (fix) after the hypothesis call
- [ ] Parse `replacement`, `unified_diff`, `caveats` from response
- [ ] Surface `Fix` in `AnalysisResult`
- [ ] `internal/analysis/ranker.go`: when LLM returns a `Fix`, attach it to top hypothesis

### Week 6 — Streaming + observability
- [ ] `internal/api/handlers_ws.go` — gorilla/websocket upgrade, progress events
- [ ] `internal/analysis/engine.go::AnalyzeStream` emits `ProgressEvent`s
- [ ] All metrics from SPEC §4.7 wired in
- [ ] `configs/prometheus/prometheus.yml`
- [ ] `configs/grafana/dashboards/sre-agent.json`
- [ ] Eval harness v2: 20 labeled cases, fix exactness (does the patched sample-app build and pass its test?)

### Deliverable
- WS endpoint streams progress; the final frame carries the full `AnalysisResult`.
- Eval shows at least one seeded bug fully patched (compiles + passes test).

---

## 4. Phase 3 — V2: deployment & polish (Weeks 7–8)

**Demo gate:** reproducible, documented, deployable. **Thesis submission.**

### Week 7 — Deployment
- [ ] `configs/k8s/*.yaml`: namespace, configmap, secret, deployment, service, ingress
- [ ] `configs/helm/sre-agent/`: Chart.yaml, values.yaml, templates/*
- [ ] `docs/DEPLOY.md`: Minikube + Kind quickstart
- [ ] `scripts/demo.sh`: up → seed bug → curl analyze → run eval → print pass/fail

### Week 8 — Polish
- [ ] `docs/API.md` — every endpoint, every field, every error code
- [ ] `docs/EVAL.md` — how to extend cases, how to interpret the report
- [ ] `README.md` — 5-minute quickstart
- [ ] Final lint, `gosec`, race-tested, coverage report
- [ ] Total Claude API spend check (target: < $50 across whole thesis)
- [ ] Pre-submission review with supervisor

### Deliverable
- `minikube start && helm install sre-agent ./configs/helm/sre-agent` works on a clean laptop.
- Eval report `tests/eval/report.md` shows ≥ 0.7 top-1, ≥ 0.9 top-3, ≥ 0.5 fix exactness.
- Total artifacts: spec, code, docs, eval, demo recording.

---

## 5. Weekly milestone summary

| Week | Phase | Milestone |
|------|-------|-----------|
| 1 | 0 | Docker image + `/healthz` |
| 2 | 1 | All three log parsers + unit tests |
| 3 | 1 | Codebase index + linker |
| 4 | 1 | End-to-end `/analyze` returning hypotheses |
| 5 | 2 | Fix suggestion in response |
| 6 | 2 | WS streaming + Prometheus metrics |
| 7 | 3 | K8s manifests + Helm chart |
| 8 | 3 | Docs, eval, demo, submission |

---

## 6. Stretch goals (only after Phase 3)

- Web UI (read-only) showing the latest analysis result
- Multi-repo codebase support
- Incremental log streaming (Kafka-style consumer)
- `claude-haiku-4` fast-path for low-severity incidents
- Result persistence (Postgres or SQLite)

These are explicitly **not** in the thesis. They exist to answer "what's
next?" if asked.

---

## 7. Risk register

| Risk | Mitigation |
|------|------------|
| Claude API drift (model deprecation, pricing change) | Pin model name in config; cache results so reruns are cheap |
| Claude non-determinism breaks eval reproducibility | Set `temperature=0`; document residual variance in `docs/EVAL.md` |
| LLM cost overruns | Token budgeting in `context.go`; cache; cap `ANTHROPIC_MAX_TOKENS` |
| Code-linker misses the right function | Eval will show this clearly; tune weights in `linker.go` |
| K8s demo fails on reviewer's laptop | Test on a fresh Minikube VM before submission; provide `scripts/demo.sh` |
| Time overrun in Week 4 (the bottleneck) | Cut WebSocket (move to Phase 3) before cutting fix generation |

---

## 8. What this plan does NOT cover (so we don't drift)

- Cloud-managed Kubernetes — out of scope
- Terraform / cloud IaC — out of scope
- Vector DBs — out of scope
- Auth beyond a static API key — out of scope
- Multi-language codebase analysis — out of scope
- Auto-deployment of suggested fixes — explicitly out of scope (this is a *suggestion* agent)

If the supervisor asks for any of these, refer them to SPECIFICATION.md §1.2
and add it as a "future work" item in the final report.
