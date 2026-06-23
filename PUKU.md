# PUKU.md

This file provides guidance to puku-cli when working with code in this repository.

## What this repo is

Documentation-only repository for a **university thesis project** (BAIUST, Bangladesh) by MD. Rifat Hossain, supervised by Golam Moktader Nayeem. The thesis is an SRE AI Agent: a Go service that ingests nginx + application logs, links errors to candidate functions in a Go codebase via `go/parser` + `go/ast` static analysis, and uses the Claude API to produce ranked root-cause hypotheses with proposed code fixes.

**Important:** the Go code, Dockerfile, Makefile, k8s manifests, etc. **do not exist yet**. They are planned across an 8-week, 4-phase roadmap in `sre-ai-agent/PLAN.md` mirroring `SPECIFICATION.md` §7. Phase 0 (Week 1) is the very first code to be written. Do not assume files like `cmd/agent/main.go`, `Dockerfile`, `go.mod`, or `Makefile` exist.

## Documentation hierarchy (read this before editing)

`SPECIFICATION.md` is the **single source of truth** — it says so explicitly in §0:

> "If anything in the other `.md` files contradicts this document, this document wins."

All other design docs are mirrors of spec sections, each carrying the same disclaimer at the top:

| File | Mirrors spec section |
|------|---------------------|
| `sre-ai-agent/ARCHITECTURE.md` | §2 (architecture, diagrams) |
| `sre-ai-agent/FOLDER_STRUCTURE.md` | §3 (repo layout) |
| `sre-ai-agent/PLAN.md` | §7 (phased roadmap) |
| `sre-ai-agent/INFRASTRUCTURE.md` | §8 (local deploy) |
| `sre-ai-agent/DEVOPS.md` | §9 (CI/CD + runbooks) |
| `sre-ai-agent/COST.md` | §10 (cost) |

**Rule:** when a mirror contradicts the spec, edit the **spec** and then update the mirror to match. Never edit a mirror to "win" an argument — it won't, and the next reader will be confused. Each mirror also has a §11 or similar "what this doc does NOT cover" block listing the cut features; respect those lists.

## Scope guardrails (explicitly out of scope — v2)

The v2 spec deliberately cut these from the v1 plan. **Do not propose adding them**, even if asked. If the supervisor pushes for any of these, the spec says: add as "future work", not v1.

- Cloud-managed Kubernetes (EKS/GKE/AKS), Terraform, IaC
- Multi-cluster, multi-region
- Service mesh (Istio/Linkerd)
- Vector databases (Pinecone, Weaviate, Qdrant, pgvector, LanceDB)
- Production observability stacks (Loki, Tempo, ELK, Datadog)
- PagerDuty / on-call integration
- Auto-remediation (the agent **suggests** fixes; it does not apply them)
- Auth beyond a single static `X-API-Key` header
- Languages other than Go for the target codebase
- Long-term log retention / archival
- External secret operators (Vault, External Secrets Operator)
- Distributed tracing (OpenTelemetry, Jaeger)
- Chaos engineering (Litmus, ChaosMesh)
- GitOps (ArgoCD, Flux)

## Stale doc warning

`sre-ai-agent/REVIEW.md` is a **2026-05-19 review of the v1 docs** (pre-replan). It references cut features (Terraform, AWS EKS, Istio, multi-cluster, chaos engineering, Cosign, Redis-based rate limiting, AWS Secrets Manager). **Do not act on it** — it is historical. The v2 spec is the authority.

## Tech stack (pinned versions — do not improvise)

When writing Phase 0+ code, use these versions. Spec §1 of `sre-ai-agent/ARCHITECTURE.md` is the canonical table.

| Component | Version | Notes |
|-----------|---------|-------|
| Go | 1.22+ | required for `go/parser` / `go/ast` |
| Gin | v1.10+ | web framework |
| `anthropic-sdk-go` | latest | only outbound service dep |
| `zap` | v1.27+ | structured logging |
| `prometheus/client_golang` | v1.20+ | metrics |
| `gorilla/websocket` | v1.5+ | WS streaming |
| `caarlos0/env` | v11+ | env loader |
| `stretchr/testify` | v1.9+ | tests |
| Docker | 24+ | multi-stage builds |
| Kubernetes | 1.28+ | local only via Minikube/Kind |
| Helm | 3.14+ | one chart |
| `golangci-lint` | v1.59+ | linter |
| `gosec` | latest | security scan |

**No external runtime services** in v1: no Postgres, no Redis, no S3, no MinIO, no vector DB. The only outbound service the agent depends on is the Claude API.

## Configuration convention

Spec §4.1 defines env vars **all prefixed `SRE_AGENT_`** (e.g. `SRE_AGENT_ANTHROPIC_API_KEY`, `SRE_AGENT_PORT`, `SRE_AGENT_LOG_LEVEL`). However, the existing `sre-ai-agent/.env.example` uses **unprefixed** names (e.g. `ANTHROPIC_API_KEY`, `API_PORT`). This is an inconsistency: trust the spec (`SRE_AGENT_*`) when writing `.env.example`, Go config, k8s manifests, Helm values, docker-compose, and CI. Update the existing `.env.example` to match the spec prefix convention.

## API surface (canonical)

From spec §4.5:

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/v1/analyze` | Synchronous analysis |
| `GET`  | `/api/v1/hypotheses/:id` | Fetch cached result |
| `GET`  | `/api/v1/incidents/:id` | Fetch one incident's detail |
| `GET`  | `/api/v1/healthz` | Liveness |
| `GET`  | `/api/v1/readyz` | Readiness (cache warm + Claude client init) |
| `GET`  | `/metrics` | Prometheus |
| `WS`   | `/api/v1/stream` | Streaming analysis |

All endpoints are prefixed `/api/v1/`. Health endpoints are `healthz`/`readyz` (not `/health`, `/health/ready`).

## Reference files (import with `@path` when needed)

These are the files to read instead of duplicating content into PUKU.md. They change frequently — import by path so puku-cli reads the current version.

- `@SPECIFICATION.md` — full spec (architecture, components, API, prompts, eval, deployment, acceptance)
- `@README.md` — 5-minute quickstart
- `@sre-ai-agent/PLAN.md` — week-by-week task checklist
- `@sre-ai-agent/INFRASTRUCTURE.md` — Dockerfile + docker-compose + K8s/Helm bring-up
- `@sre-ai-agent/DEVOPS.md` — CI/CD pipeline, runbooks, security baseline
- `@sre-ai-agent/.env.example` — env var template (currently inconsistent with spec prefix; see "Configuration convention" above)
- `@sre-ai-agent/FOLDER_STRUCTURE.md` — target repo layout for the code that will be written in Phase 0

## "Done" means

Per spec §12 acceptance criteria — the thesis is "done" when **all** of these are true:

- `make build` succeeds on a clean checkout
- `make test` passes (unit + integration + eval)
- `docker compose up` brings the system up end-to-end on a laptop
- `helm install sre-agent ./configs/helm/sre-agent` works against a fresh `minikube start`
- `POST /api/v1/analyze` against a seeded bug returns a hypothesis whose `suspect_function` matches ground truth for ≥ 70% of eval cases
- For at least one seeded bug, the returned `fix.replacement` compiles and makes the test pass
- `/metrics` exposes all metrics listed in spec §4.7
- `docs/API.md`, `docs/EVAL.md`, `docs/DEPLOY.md` exist and are accurate
- `README.md` has a 5-minute quickstart
- Total Claude API spend across all development + evaluation is < $50 USD

Top-1 accuracy target ≥ 0.7, top-3 ≥ 0.9, fix exactness ≥ 0.5.

## Conventions and gotchas

- **Log format coverage:** the agent must parse nginx access (combined format), nginx error, and Go app JSON logs (with optional stack traces). Do not propose adding other log sources in v1.
- **Target codebase is Go only.** Multi-language support is out of scope.
- **Suggestion agent, not remediation agent.** The `fix.replacement` field is a proposal. Never add auto-apply.
- **Codebase is mounted read-only** in the container; the agent only reads it.
- **AST cache is in-process / on-disk JSON** at `SRE_AGENT_CODEBASE_CACHE_DIR` (default `/tmp/sre-agent/cache`). No database.
- **LLM temperature must be `0`** for eval reproducibility (DEVOPS §7.3).
- **License:** MIT.
- **Commit messages and PR style** are not documented in the repo — match the existing git history (`git log`) for tone, which is sparse so far.
- **Git remote:** `github.com/rifatbond007/Shallow-plan-for-SRE-AGENT` — `gh` CLI is available and appropriate.
