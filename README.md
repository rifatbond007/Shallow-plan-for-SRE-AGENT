# SRE AI Agent

An AI-powered Site Reliability Engineering agent that, given operational logs and a Go codebase, **detects incidents → ranks root-cause hypotheses → proposes code-level fixes**.

> **Read [`SPECIFICATION.md`](./SPECIFICATION.md) first.** It is the single source of truth for this project. Every other document mirrors a section of it.

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
| **[`SPECIFICATION.md`](./SPECIFICATION.md)** | **Single source of truth.** Scope, architecture, components, API, prompts, eval, deployment, acceptance criteria. |
| [`sre-ai-agent/ARCHITECTURE.md`](./sre-ai-agent/ARCHITECTURE.md) | Diagrams only. Mirrors spec §2. |
| [`sre-ai-agent/FOLDER_STRUCTURE.md`](./sre-ai-agent/FOLDER_STRUCTURE.md) | Repo layout. Mirrors spec §3. |
| [`sre-ai-agent/PLAN.md`](./sre-ai-agent/PLAN.md) | Week-by-week implementation plan. Mirrors spec §7. |
| [`sre-ai-agent/INFRASTRUCTURE.md`](./sre-ai-agent/INFRASTRUCTURE.md) | Local deployment (Docker + Minikube/Kind + Helm). Mirrors spec §8. |
| [`sre-ai-agent/DEVOPS.md`](./sre-ai-agent/DEVOPS.md) | CI/CD, security, runbooks. Mirrors spec §9. |
| [`sre-ai-agent/COST.md`](./sre-ai-agent/COST.md) | Cost reality check. Mirrors spec §10. |
| [`PROPOSAL_ENHANCED.md`](./PROPOSAL_ENHANCED.md) | Academic proposal (for the university submission). |
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

These are conscious cuts to keep the thesis scope defensible. See [`SPECIFICATION.md` §1.2](./SPECIFICATION.md#12-non-goals-explicitly-out-of-scope).

## Architecture

See [`SPECIFICATION.md` §2](./SPECIFICATION.md#2-high-level-architecture) and [`sre-ai-agent/ARCHITECTURE.md`](./sre-ai-agent/ARCHITECTURE.md) for the full diagram and component map.

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

**Explicitly not used** (kept out of scope): Terraform, Istio, vector DBs, ELK/Loki/Tempo, PagerDuty, cloud-managed K8s. See [`SPECIFICATION.md` §1.2](./SPECIFICATION.md#12-non-goals-explicitly-out-of-scope).

## API surface (summary)

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/v1/analyze` | Submit logs + codebase path; get back incidents, hypotheses, and a fix |
| `GET`  | `/api/v1/hypotheses/:id` | Fetch a cached analysis result |
| `WS`   | `/api/v1/stream` | Streaming analysis with progress events |
| `GET`  | `/api/v1/healthz`, `/api/v1/readyz` | Liveness / readiness |
| `GET`  | `/metrics` | Prometheus scrape endpoint |

Full contract: [`SPECIFICATION.md` §4.5, §6](./SPECIFICATION.md) and (later) `docs/API.md`.

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

Full methodology: [`SPECIFICATION.md` §9](./SPECIFICATION.md#9-evaluation-methodology-a-thesis-is-judged-on-this).

## Cost

**$0/month** infrastructure. **~$10–50 total** Claude API across the whole
thesis. See [`COST.md`](./sre-ai-agent/COST.md).

## Documentation map

| Document | Purpose |
|----------|---------|
| **[`SPECIFICATION.md`](./SPECIFICATION.md)** | **Single source of truth.** |
| [`sre-ai-agent/ARCHITECTURE.md`](./sre-ai-agent/ARCHITECTURE.md) | Diagrams (mirror of spec §2). |
| [`sre-ai-agent/FOLDER_STRUCTURE.md`](./sre-ai-agent/FOLDER_STRUCTURE.md) | Repo layout (mirror of spec §3). |
| [`sre-ai-agent/PLAN.md`](./sre-ai-agent/PLAN.md) | Week-by-week plan (mirror of spec §7). |
| [`sre-ai-agent/INFRASTRUCTURE.md`](./sre-ai-agent/INFRASTRUCTURE.md) | Local deploy (mirror of spec §8). |
| [`sre-ai-agent/DEVOPS.md`](./sre-ai-agent/DEVOPS.md) | CI/CD + runbooks (mirror of spec §9). |
| [`sre-ai-agent/COST.md`](./sre-ai-agent/COST.md) | Cost (mirror of spec §10). |
| [`PROPOSAL_ENHANCED.md`](./PROPOSAL_ENHANCED.md) | Academic proposal (for the university submission). |

## License

MIT
