# SRE AI Agent - Architecture (v2)

> **Source of truth:** [`SPECIFICATION.md`](./SPECIFICATION.md), §2.
> This document mirrors the architecture section of the spec with diagrams only.
> If anything here contradicts the spec, the spec wins.

---

## 1. One-line description

An agent that, given operational logs and a Go codebase, detects incidents,
ranks root-cause hypotheses, and proposes code-level fixes — driven by the
Claude API with deterministic pre/post-processing.

---

## 2. System context

```
                            External
   ┌────────────────────┐  ┌────────────────────┐
   │  Operator / CI /   │  │  Claude API        │
   │  Web UI            │  │  (Anthropic)       │
   └──────────┬─────────┘  └──────────▲─────────┘
              │                       │
              │  HTTP/WS             │  HTTPS
              ▼                       │
   ┌──────────────────────────────────┴─────────┐
   │              SRE AI Agent (this repo)       │
   │                                              │
   │  Ingest  →  Codebase  →  Analysis  →  API    │
   └──────────────────────────────────────────────┘
              ▲
              │  mounted read-only
   ┌──────────┴─────────┐
   │  Target Go source  │
   │  + log files       │
   └────────────────────┘
```

---

## 3. Component diagram

```
┌────────────────────────────────────────────────────────────────────────┐
│                          SRE AI AGENT                                   │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│   ┌──────────────┐    ┌──────────────────┐    ┌─────────────────────┐  │
│   │  Data        │    │  Ingestion       │    │  Codebase           │  │
│   │  Sources     │───▶│  Layer           │    │  Analyzer           │  │
│   │              │    │                  │    │                     │  │
│   │ • nginx      │    │ • nginx_access   │    │ • scanner.go        │  │
│   │ • nginx err  │    │ • nginx_error    │    │ • ast.go            │  │
│   │ • app JSON   │    │ • app_json       │    │ • callgraph.go      │  │
│   │ • go source  │    │ • normalizer     │    │ • linker.go         │  │
│   └──────────────┘    │ • incident.go    │    └──────────┬──────────┘  │
│                       └────────┬─────────┘               │             │
│                                │                         │             │
│                                ▼                         ▼             │
│                       ┌──────────────────────────────────────┐         │
│                       │         Analysis Engine              │         │
│                       │  • engine.go (orchestrator)          │         │
│                       │  • patterns.go (deterministic)       │         │
│                       │  • context.go (prompt builder)       │         │
│                       │  • claude.go (LLM client)            │         │
│                       │  • ranker.go (score fusion)          │         │
│                       └─────────────────┬────────────────────┘         │
│                                         │                              │
│                                         ▼                              │
│                       ┌──────────────────────────────────────┐         │
│                       │         API Server (Gin)              │         │
│                       │  • REST   POST /api/v1/analyze        │         │
│                       │  • GET    /api/v1/hypotheses/:id     │         │
│                       │  • WS     /api/v1/stream             │         │
│                       │  • GET    /api/v1/healthz, /readyz   │         │
│                       │  • GET    /metrics                    │         │
│                       └─────────────────┬────────────────────┘         │
│                                         │                              │
│                                         ▼                              │
│                       ┌──────────────────────────────────────┐         │
│                       │         Storage (in-process)          │         │
│                       │  • cache.go (LRU + TTL)               │         │
│                       │  • store.go (analysis results)        │         │
│                       └──────────────────────────────────────┘         │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

External integrations:

- **Claude API** (Anthropic) — the only outbound service the agent depends on.
- **Prometheus** (optional) — scrapes `/metrics`. Self-hosted or via kube-prometheus-stack.

There are **no** other external dependencies (no Postgres, Redis, S3, vector DB,
Loki, Tempo, Jaeger, etc.) in v1. The data flow is intentionally minimal.

---

## 4. Request lifecycle

The agent's value is in this loop. Every component exists to make it fast
and explainable.

```
HTTP POST /api/v1/analyze  { logs, codebase_path, top_k }
   │
   ▼
[1] Validate (size, paths) ──────────────▶ 400 / 413
   │
   ▼
[2] Parse logs (3 parsers in parallel)
   │
   ▼
[3] Normalize → []LogEntry
   │
   ▼
[4] Group → []Incident   (sliding window + similarity)
   │
   ▼
[5] Codebase analysis (one-shot, disk-cached)
   │   • AST, call graph, function index
   │   • load from /tmp/sre-agent/cache if hash matches
   ▼
[6] For each incident:
       a. pattern matcher  → []PatternMatch
       b. code-linker      → []ScoredFunction (top-K)
       c. context builder  → LLM prompt (system + incident + code)
       d. Claude call      → ranked hypotheses (JSON)
       e. ranker           → merge pattern + LLM confidence
   │
   ▼
[7] For top hypothesis: Claude call → Fix { replacement, diff, caveats }
   │
   ▼
[8] Cache AnalysisResult
   │
   ▼
HTTP 200  AnalysisResult
```

For long-running analysis, the same loop emits events over WebSocket:

```
→ { type: "progress", stage: "parsing",     pct: 10 }
→ { type: "progress", stage: "codebase",    pct: 25 }
→ { type: "incident", data: {...} }
→ { type: "hypothesis", data: {...} }
→ { type: "fix", data: {...} }
→ { type: "done", result_id: "..." }
```

---

## 5. Data flow (high level)

```
┌──────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ Logs │────▶│ Ingest   │────▶│ Codebase │────▶│ Analysis │────▶│ Response │
│      │     │          │     │  Index   │     │  Engine  │     │          │
└──────┘     └──────────┘     └──────────┘     └──────────┘     └──────────┘
              parse +          AST +            pattern +          JSON
              group            call graph       LLM                result
```

---

## 6. Component responsibilities

| Component | Responsibility | Spec section |
|-----------|----------------|--------------|
| `ingest/nginx_access.go` | Parse combined log format | §4.2 |
| `ingest/nginx_error.go`  | Parse nginx error lines   | §4.2 |
| `ingest/app_json.go`     | Parse JSON app logs       | §4.2 |
| `ingest/normalizer.go`   | Unified `LogEntry`        | §4.2 |
| `ingest/incident.go`     | Time-window clustering    | §4.2 |
| `codebase/scanner.go`    | Walk Go source tree       | §4.3 |
| `codebase/ast.go`        | `go/parser` + `go/ast`    | §4.3 |
| `codebase/callgraph.go`  | Function call graph       | §4.3 |
| `codebase/linker.go`     | Log error → function map  | §4.3 |
| `analysis/patterns.go`   | Deterministic signatures  | §4.4 |
| `analysis/context.go`    | Build LLM prompt          | §4.4 |
| `analysis/claude.go`     | Anthropic SDK wrapper     | §4.4 |
| `analysis/ranker.go`     | Score fusion              | §4.4 |
| `analysis/engine.go`     | Orchestrator              | §4.4 |
| `api/handlers_*`         | HTTP handlers             | §4.5 |
| `storage/cache.go`       | LRU + TTL cache           | §4.6 |
| `pkg/logger`             | Structured logging        | §4.7 |
| `pkg/metrics`            | Prometheus collectors     | §4.7 |

---

## 7. Technology stack (pinned)

| Layer | Technology | Version | Why |
|-------|------------|---------|-----|
| Language | Go | 1.22+ | Static analysis, K8s-friendly |
| Web framework | Gin | v1.10+ | Most-used, fast |
| LLM SDK | `anthropic-sdk-go` | latest | Official |
| Logging | `zap` | v1.27+ | Fastest structured logger |
| Metrics | `prometheus/client_golang` | v1.20+ | Standard |
| WebSocket | `gorilla/websocket` | v1.5+ | Reliable |
| Testing | `stretchr/testify` | v1.9+ | Idiomatic Go tests |
| Config | `caarlos0/env` | v11+ | Simple env loader |
| Cache | custom (no dep) | — | Avoid a Redis dep |
| Container | Docker | 24+ | Multi-stage builds |
| Orchestration | Kubernetes | 1.28+ | Local via Minikube/Kind |
| Package manager | Helm | 3.14+ | For the chart |
| CI | GitHub Actions | — | Free for the thesis |

Explicitly **not** used (to keep the thesis scope tight):
- Terraform, AWS/GCP/Azure-specific manifests
- Istio / Linkerd
- Prometheus Operator (use plain manifests in the demo)
- ELK, Loki, Tempo, Jaeger
- Postgres, Redis, S3, MinIO
- Vector DBs (pgvector, Pinecone, Weaviate, Qdrant, LanceDB)

---

## 8. Why this architecture is "small enough to defend"

A thesis reviewer will ask: "Why a whole new service? Why not just a script?"

Three honest answers this design supports:

1. **The lifecycle matters.** Detect → hypothesize → suggest fix is a *pipeline*
   with distinct, testable stages. The architecture makes those stages first-class.
2. **Code context is the novel bit.** Static code analysis linked to log errors
   is the contribution. The `codebase/` package exists to make that link
   explicit and measurable.
3. **Reproducibility is a deliverable.** A versioned API, a labeled eval set,
   and a Helm chart let a reviewer rerun everything on a laptop.

Anything that doesn't serve one of those three is cut.
