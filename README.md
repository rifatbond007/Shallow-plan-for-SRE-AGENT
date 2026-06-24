# SRE AI Agent

An AI-powered Site Reliability Engineering agent that, given operational logs and a Go codebase, **detects incidents → ranks root-cause hypotheses → proposes code-level fixes**.

> **Read [`SPECIFICATION.md`](./docs/SPECIFICATION.md) first.** It is the single source of truth for this project. Every other document mirrors a section of it.

## What this project does (one sentence)

Reads nginx + Go app logs, links the errors to candidate functions in a Go codebase via static analysis, asks Claude to reason about the cause, and returns a ranked list of hypotheses with a proposed code fix for the top one.

## Quick start (5 minutes)

```bash
# 1. Clone & enter
git clone <repo> && cd sre-ai-agent

# 2. Configure
cp .env.example .env
# Edit .env: set SRE_AGENT_ANTHROPIC_API_KEY=sk-ant-...

# 3. Run
cd configs && docker compose up -d

# 4. Verify
curl http://localhost:8080/api/v1/healthz
# → {"status":"ok"}
```

Then `POST /api/v1/analyze` with `{ "logs": "...", "codebase_path": "/codebase" }` and you get a JSON response with `incidents`, `hypotheses`, and `fixes`.

Full contract: [`docs/API.md`](./docs/API.md) (to be written in Phase 3).
Full deploy guide: [`docs/DEPLOY.md`](./docs/DEPLOY.md) (to be written in Phase 3).

## Documentation map

| Document | Purpose |
|----------|---------|
| **[`SPECIFICATION.md`](./docs/SPECIFICATION.md)** | **Single source of truth.** Scope, architecture, components, API, prompts, eval, deployment, acceptance criteria. |
| [`sre-ai-agent/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) | Diagrams only. Mirrors spec §2. |
| [`sre-ai-agent/FOLDER_STRUCTURE.md`](./docs/FOLDER_STRUCTURE.md) | Repo layout. Mirrors spec §3. |
| [`sre-ai-agent/PLAN.md`](./docs/PLAN.md) | Week-by-week implementation plan. Mirrors spec §7. |
| [`sre-ai-agent/INFRASTRUCTURE.md`](./docs/INFRASTRUCTURE.md) | Local deployment (Docker + Minikube/Kind + Helm). Mirrors spec §8. |
| [`sre-ai-agent/DEVOPS.md`](./docs/DEVOPS.md) | CI/CD, security, runbooks. Mirrors spec §9. |
| [`sre-ai-agent/COST.md`](./docs/COST.md) | Cost reality check. Mirrors spec §10. |
| `docs/API.md` | REST + WebSocket contract. *(Phase 3)* |
| `docs/EVAL.md` | How to run and read the evaluation. *(Phase 3)* |
| `docs/DEPLOY.md` | Step-by-step deployment walkthrough. *(Phase 3)* |

## What this project is NOT

- Not a cloud-deployed, multi-region production system
- Not a Terraform-managed infrastructure
- Not a vector database + RAG platform
- Not a multi-language static analyser
- Not a service mesh demo
- Not an auto-remediation agent (it **suggests** fixes; it does not apply them)

These are conscious cuts to keep the thesis scope defensible. See [`SPECIFICATION.md` §1.2](./docs/SPECIFICATION.md#12-non-goals-explicitly-out-of-scope).

## Research gap addressed by this work

The SRE × LLM literature is dominated by **production-scale, multi-agent frameworks** (e.g. Ant Group's OpenDerisk, [arXiv:2510.13561](https://arxiv.org/abs/2510.13561)) and **human-agent collaborative SRE teams** (Madduri, 2026, *Power System Protection and Control*, 54(1):414–423). Both report strong industrial numbers — 67% autonomous resolution, 80% MTTR reduction, 60,000 runs/day, 3,000 daily users — but they share gaps that this thesis explicitly targets:

| # | Gap in the prior literature | What this project does differently |
|---|---|---|
| **1** | **No static code analysis integrated with LLM RCA.** OpenDerisk's *Code-Agent* reads runtime artefacts; the Agentic SRE paper is log-only. Neither builds a `go/parser` + `go/ast` function index, call graph, or stack-trace-aware linker. | This project's **core contribution** is the `internal/codebase` package (spec §4.3): a real AST + intra-package call graph + scored candidate-function linker that gives the LLM a *curated, top-K function context* for every incident. The linker is explainable — every `ScoredFunction` carries the `Reasons` for its score (name match, doc match, stack-trace frame match, callee proximity). |
| **2** | **No reproducible, labeled evaluation harness.** Both papers report production numbers from private datasets (AntRCA: 1,743 cases, 6,000+ real-world cases). | Spec §9 mandates `tests/eval/cases.json` with labeled `ground_truth` (function, fix_summary) and quantitative targets — **top-1 ≥ 0.7, top-3 ≥ 0.9, fix exactness ≥ 0.5, p95 < 30 s** — runnable on a laptop with `make eval`. The eval dataset is **committed**, so reviewers can rerun and reproduce. |
| **3** | **No function-level *code fix* as a deliverable.** OpenDerisk produces "Handling Opinions" and "Smart Testing"; the Agentic SRE paper stops at "Resolution Verified." Neither generates a concrete replacement function or unified diff. | Spec §4.4 + §5.3 generate a `Fix` with `Replacement` (full new function body) and `UnifiedDiff`, then **verify it compiles and passes the seeded test** — a *fix-exactness* metric no other SRE-LLM paper reports. |
| **4** | **No hybrid deterministic + LLM pipeline.** Both frameworks rely on the LLM end-to-end. OpenDerisk explicitly admits an **accuracy-latency trade-off** (V3 framework: 22 min vs 6 min for V1 baseline). | Spec §4.4 `patterns.go` runs a **deterministic regex signature matcher first**. On a high-confidence pattern hit (`> 0.9`), the LLM call is shortened or skipped, and the matched hypothesis is **boosted to rank 1** by `ranker.go`. This is both a *latency lever* and an *explainability anchor* for the LLM. |
| **5** | **No structured, auditable evidence trail per hypothesis.** OpenDerisk has a "Reasoning Flow / Evidence Chain" UI; the Agentic SRE paper is implicit. Neither ties every claim to a specific log entry ID, code ref, and score. | Each `Hypothesis` (spec §4.4) carries a typed `[]Evidence` array (`log` / `code` / `pattern`), and each `ScoredFunction` carries the `Reasons` it was chosen. Every rank decision is reconstructable. |
| **6** | **Closed / vendor-tied ecosystems.** OpenDerisk is Python + Ant Group's private observability stack; the Agentic SRE paper is a vendor framework. | This project is **Go 1.22+**, single binary, MIT-licensed, runs end-to-end on a laptop with `docker compose` (spec §8). No external runtime services. |
| **7** | **No cost/energy reporting.** Neither paper discusses token cost or carbon. | Spec §10 + `COST.md` enforce a hard ceiling of **< $50 USD total Claude API spend** for the entire thesis. A useful counter-example to industrial-scale SRE agents. |
| **8** | **Scope is not thesis-defensible.** 13 specialists, 50+ agents, 3,000 daily users, 60,000 runs/day are not reproducible by an external examiner. | The v2 spec explicitly **cuts** Terraform, AWS/cloud, vector DBs, multi-cluster, service mesh, ELK, and auto-remediation (spec §1.2). A reviewer can `git clone` → `docker compose up` → `make eval` in under 10 minutes (DEVOPS §8). |

**Positioning statement.** Where OpenDerisk optimizes for *industrial scale* and the Agentic SRE paper optimizes for *operational collaboration*, this project optimizes for **explainability, reproducibility, and the closed loop from runtime error → function in the code → proposed fix → test-validated fix** in a single, defensible artifact.

## Architecture

See [`SPECIFICATION.md` §2](./docs/SPECIFICATION.md#2-high-level-architecture) and [`sre-ai-agent/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) for the full diagram and component map.

In short: **Ingest → Codebase → Analysis → API**, with Claude as the only external dependency.

## Tech Stack (pinned)

| Component | Technology | Version | Why |
|-----------|------------|---------|-----|
| Language | Go | 1.22+ | `go/parser` / `go/ast` for static analysis |
| Web framework | Gin | v1.10+ | De facto Go web framework |
| LLM SDK | `anthropic-sdk-go` | latest | Official |
| Logging | `zap` | v1.27+ | Fast structured logs |
| Metrics | `prometheus/client_golang` | v1.20+ | Standard |
| WebSocket | `gorilla/websocket` | v1.5+ | Reliable |
| Config | `caarlos0/env` | v11+ | Simple env loader |
| Container | Docker | 24+ | Multi-stage builds |
| Orchestration | Kubernetes | 1.28+ | Local via Minikube/Kind |
| Package manager | Helm | 3.14+ | One chart, multiple values |
| CI | GitHub Actions | — | Free for the thesis |

**Explicitly not used** (kept out of scope): Terraform, Istio, vector DBs, ELK/Loki/Tempo, PagerDuty, cloud-managed K8s. See [`SPECIFICATION.md` §1.2](./docs/SPECIFICATION.md#12-non-goals-explicitly-out-of-scope).

## API surface (summary)

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/v1/analyze` | Submit logs + codebase path; get back incidents, hypotheses, and a fix |
| `GET`  | `/api/v1/hypotheses/:id` | Fetch a cached analysis result |
| `WS`   | `/api/v1/stream` | Streaming analysis with progress events |
| `GET`  | `/api/v1/healthz`, `/api/v1/readyz` | Liveness / readiness |
| `GET`  | `/metrics` | Prometheus scrape endpoint |

Full contract: [`SPECIFICATION.md` §4.5, §6](./docs/SPECIFICATION.md) and (later) `docs/API.md`.

## Quick start (5 minutes)

```bash
# From repo root
git clone <repo> && cd Shallow-plan-for-SRE-AGENT

# Configure
cp sre-ai-agent/.env.example sre-ai-agent/.env
# Edit .env: set SRE_AGENT_ANTHROPIC_API_KEY

# Run
cd sre-ai-agent/configs
docker compose up -d

# Verify
curl http://localhost:8080/api/v1/healthz
```

## Local Kubernetes (optional, for the deployable demo)

```bash
# Build into Minikube's Docker daemon
eval $(minikube docker-env)
cd sre-ai-agent
docker build -t sre-agent:dev .

helm install sre-agent ./configs/helm/sre-agent \
  --namespace sre-agent --create-namespace

kubectl -n sre-agent port-forward svc/sre-agent 8080:8080
```

## Evaluation

`make eval` runs the labeled case set in `sre-ai-agent/tests/eval/cases.json`
and writes `sre-ai-agent/tests/eval/report.md`. Target: ≥ 0.7 top-1 accuracy,
≥ 0.9 top-3 accuracy, ≥ 0.5 fix exactness.

Full methodology: [`SPECIFICATION.md` §9](./docs/SPECIFICATION.md#9-evaluation-methodology-a-thesis-is-judged-on-this).

## Cost

**$0/month** infrastructure. **~$10–50 total** Claude API across the whole
thesis. See [`COST.md`](./docs/COST.md).

## License

MIT
