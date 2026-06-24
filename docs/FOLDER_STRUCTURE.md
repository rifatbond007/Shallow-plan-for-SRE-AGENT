# SRE AI Agent - Folder Structure (v2)

> **Source of truth:** [`SPECIFICATION.md`](./SPECIFICATION.md), §3.
> This file is a copy of §3 with annotations for what each path holds.
> If anything here contradicts the spec, the spec wins.

---

## 1. Top-level layout

```
repo-root/
├── README.md
├── docs/
│   ├── AGENTS.md
│   ├── SPECIFICATION.md           # ← single source of truth
│   ├── ARCHITECTURE.md            # mirrors SPECIFICATION.md §2
│   ├── FOLDER_STRUCTURE.md        # this file
│   ├── PLAN.md                    # mirrors SPECIFICATION.md §7
│   ├── INFRASTRUCTURE.md          # mirrors SPECIFICATION.md §8 (local only)
│   ├── DEVOPS.md                  # mirrors SPECIFICATION.md §9 (trimmed)
│   ├── COST.md                    # mirrors SPECIFICATION.md §10
│   ├── API.md                     # Phase 3
│   ├── EVAL.md                    # Phase 3
│   └── DEPLOY.md                  # Phase 3
├── papers/
└── sre-ai-agent/
    ├── cmd/                  # entrypoints (one binary: agent)
    ├── internal/             # private application code
    ├── pkg/                  # small reusable packages
    ├── prompts/              # LLM prompt templates (text files, shipped)
    ├── configs/              # docker-compose, k8s manifests, helm chart
    ├── tests/                # unit / integration / evaluation data
    ├── scripts/              # dev/demo helpers
    ├── go.mod
    ├── go.sum
    ├── Makefile
    ├── Dockerfile
    └── .env.example
```

---

## 2. Detailed tree

```
sre-ai-agent/
│
├── cmd/
│   └── agent/
│       └── main.go                 # entrypoint, wires everything
│
├── internal/
│   │
│   ├── config/
│   │   └── config.go               # env loading, validation
│   │
│   ├── ingest/
│   │   ├── ingest.go               # Parser / Normalizer / Grouper interfaces
│   │   ├── nginx_access.go         # combined-log-format
│   │   ├── nginx_error.go          # nginx error format
│   │   ├── app_json.go             # JSON application logs
│   │   ├── normalizer.go           # LogEntry unified type
│   │   └── incident.go             # time-window clustering
│   │
│   ├── codebase/
│   │   ├── scanner.go              # walk Go source tree
│   │   ├── ast.go                  # go/parser + go/ast
│   │   ├── callgraph.go            # intra-package call graph
│   │   └── linker.go               # log error → function scoring
│   │
│   ├── analysis/
│   │   ├── types.go                # Hypothesis, Fix, AnalysisResult
│   │   ├── engine.go               # orchestrator
│   │   ├── patterns.go             # deterministic signature matcher
│   │   ├── context.go              # builds LLM prompt
│   │   ├── claude.go               # Anthropic SDK wrapper
│   │   └── ranker.go               # combines pattern + LLM scores
│   │
│   ├── api/
│   │   ├── server.go               # gin engine, middleware
│   │   ├── handlers_analyze.go     # POST /api/v1/analyze
│   │   ├── handlers_get.go         # GET /api/v1/hypotheses/:id
│   │   ├── handlers_ws.go          # WS /api/v1/stream
│   │   ├── handlers_meta.go        # /healthz, /readyz, /metrics
│   │   └── errors.go               # error → HTTP code mapping
│   │
│   └── storage/
│       ├── cache.go                # LRU + TTL
│       └── store.go                # analysis result store
│
├── pkg/
│   │
│   ├── logger/
│   │   └── logger.go               # zap wrapper
│   │
│   └── metrics/
│       └── metrics.go              # prometheus collectors
│
├── prompts/
│   ├── system.txt                  # base system prompt
│   ├── hypothesis.txt              # ranked hypotheses (JSON)
│   └── fix.txt                     # proposed code fix (JSON)
│
├── configs/
│   │
│   ├── docker-compose.yml          # sre-agent + prometheus + grafana
│   ├── docker-compose.dev.yml      # dev overrides (verbose logs, sample-app)
│   │
│   ├── k8s/
│   │   ├── namespace.yaml
│   │   ├── configmap.yaml
│   │   ├── secret.yaml             # ANTHROPIC_API_KEY
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml            # optional
│   │
│   ├── helm/
│   │   └── sre-agent/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       ├── values-dev.yaml
│   │       └── templates/
│   │           ├── deployment.yaml
│   │           ├── service.yaml
│   │           ├── configmap.yaml
│   │           ├── secret.yaml
│   │           └── ingress.yaml
│   │
│   ├── prometheus/
│   │   └── prometheus.yml          # scrape config for /metrics
│   │
│   └── grafana/
│       └── dashboards/
│           └── sre-agent.json      # provisioned dashboard
│
├── tests/
│   │
│   ├── data/
│   │   ├── logs/
│   │   │   ├── nginx-access.log
│   │   │   ├── nginx-error.log
│   │   │   └── app.log
│   │   └── code/
│   │       └── sample-app/         # seeded Go service with 3 known bugs
│   │
│   ├── unit/                       # *_test.go colocated is also OK
│   │
│   ├── integration/
│   │   └── api_test.go             # POST /analyze end-to-end (uses mocks)
│   │
│   └── eval/
│       ├── cases.json              # labeled incidents
│       ├── runner.go               # runs all cases
│       └── report.go               # accuracy, top-K, F1
│
├── scripts/
│   ├── gen_sample_logs.sh
│   ├── seed_bugs.sh                # mutates sample-app
│   └── demo.sh                     # full demo: up → curl → eval
│
├── docs/
│   ├── API.md                      # REST + WS contract
│   ├── EVAL.md                     # how to run and read the eval
│   └── DEPLOY.md                   # docker-compose + minikube/helm
│
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── .env.example
```

---

## 3. What goes where — quick rules

| If you're adding... | Put it in... |
|---------------------|--------------|
| A new HTTP endpoint | `internal/api/handlers_*.go` |
| A new log parser | `internal/ingest/<source>.go` |
| A new error signature | `internal/analysis/patterns.go` |
| A new prompt section | `prompts/*.txt` (no logic in code) |
| A new Prometheus metric | `pkg/metrics/metrics.go` |
| A new env var | `internal/config/config.go` |
| A new K8s resource | `configs/k8s/<kind>.yaml` |
| A new Helm value | `configs/helm/sre-agent/values.yaml` |
| A new test case | `tests/eval/cases.json` |
| Documentation for a feature | `docs/*.md` |

---

## 4. Naming conventions

| Type | Convention | Example |
|------|------------|---------|
| Go files | lowercase_snake.go | `nginx_access.go` |
| Go test files | `<name>_test.go` | `parser_test.go` |
| Configs | lowercase-hyphen | `docker-compose.yml` |
| Markdown | UPPERCASE.md for top-level | `SPECIFICATION.md` |
| Markdown | Title-Case.md inside docs/ | `API.md` |
| Prompts | snake_case.txt | `hypothesis.txt` |

---

## 5. Out-of-tree (deliberately not committed)

- `bin/` — build output (gitignored)
- `.env` — real secrets (gitignored)
- `/tmp/sre-agent/cache/` — AST cache (gitignored; rebuilt on first run)
- `vendor/` — not used; rely on `go.sum` instead

---

## 6. What's NOT in this repo (intentionally)

The previous plan listed `terraform/`, `monitoring/`, `load-balancer.md`-style
write-ups, etc. Those are removed in v2. See `SPECIFICATION.md` §1.2 for the
full non-goal list.

If a contributor wants to add back a Terraform module or cloud-specific
Helm values, **don't**. Put it in a separate repo (`sre-agent-deploy-aws`)
under your personal account and link it from `docs/DEPLOY.md` as "out of
scope for the thesis; see this repo for an optional cloud variant."
