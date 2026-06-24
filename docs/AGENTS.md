# SRE AI Agent — Agent Guidelines

## Status: design phase (no code yet)

This repo contains specification documents only. No Go code, `go.mod`, `Makefile`, or `Dockerfile` exist yet. Implementation hasn't started.

## Getting oriented

- **`SPECIFICATION.md`** — single source of truth (1066 lines). Every other `.md` mirrors a section of it. When docs conflict, the spec wins.
- **`sre-ai-agent/`** — future Go module root. Currently only has `.env.example`.
- **`papers/`** — reference PDFs used in the literature review.

## Key facts for an implementing agent

| Fact | Detail |
|------|--------|
| Go module path | `github.com/rifatbond007/sre-ai-agent` |
| Module root | `sre-ai-agent/` (not repo root) |
| Go version | 1.22+ (needed for `go/parser`, `go/ast`) |
| Entrypoint | `sre-ai-agent/cmd/agent/main.go` |
| Planned deps | gin, anthropic-sdk-go, zap, prometheus/client_golang, gorilla/websocket, caarlos0/env |
| Env prefix | All env vars prefixed `SRE_AGENT_` (see `.env.example` and spec §4.1) |
| Single binary | No external runtime services (no Redis, Postgres, S3, vector DB) |
| Only outbound dep | Claude API (Anthropic) |

## Planned directory structure

```
sre-ai-agent/
├── cmd/agent/main.go          # wires everything
├── internal/
│   ├── config/config.go       # env loading (caarlos0/env)
│   ├── ingest/                # parsers + normalizer + incident grouper
│   ├── codebase/              # scanner, AST, call graph, linker
│   ├── analysis/              # engine, patterns, context, claude, ranker
│   ├── api/                   # gin handlers (analyze, get, ws, meta, errors)
│   └── storage/               # LRU+TTL cache (no deps)
├── pkg/
│   ├── logger/logger.go       # zap wrapper
│   └── metrics/metrics.go     # Prometheus collectors
├── prompts/                   # LLM template .txt files (shipped, not code)
├── configs/                   # docker-compose, k8s manifests, helm, prometheus, grafana
├── tests/
│   ├── data/logs/             # fixture log files
│   ├── data/code/sample-app/  # small Go app with 3 seeded bugs (nil ptr, timeout, SQLi)
│   ├── unit/
│   ├── integration/api_test.go
│   └── eval/                  # cases.json, runner.go, report.go
├── scripts/                   # gen_sample_logs.sh, seed_bugs.sh, demo.sh
├── go.mod / go.sum
├── Makefile
└── Dockerfile (multi-stage, golang:1.22-alpine → alpine:3.20, non-root)
```

## Conventions (from FOLDER_STRUCTURE.md)

- Go files: `lowercase_snake.go`
- Configs: `lowercase-hyphen` (e.g. `docker-compose.yml`)
- Top-level markdown: `UPPERCASE.md`
- `docs/` markdown: `Title-Case.md`
- Prompts: `snake_case.txt`
- No `vendor/` dir — use `go.sum`
- Generated files (`// Code generated ... DO NOT EDIT.`) skipped by AST scanner

## Planned commands (to create in Phase 0)

```
make build        # go build ./cmd/agent
make test         # go test ./... (unit + integration + eval)
make run          # go run ./cmd/agent
make docker-build # docker build -t sre-agent:dev .
make docker-run   # docker compose up -d
make lint         # golangci-lint run ./...
make eval         # go run ./tests/eval/runner.go → tests/eval/report.md
```

## Implementation phases (from spec §7)

| Phase | Weeks | What it produces |
|-------|-------|-----------------|
| 0 | 1 | Scaffold: main.go with /healthz, Docker image, Makefile |
| 1 | 2–4 | Log parsers, codebase index, Claude hypothesis → POST /analyze |
| 2 | 5–6 | Fix generation, WebSocket streaming, Prometheus metrics |
| 3 | 7–8 | K8s manifests, Helm chart, docs, 20+ eval cases |

Demo gate for each phase:
- P0: `docker compose up` → `curl :8080/healthz` returns 200
- P1: POST /analyze returns ranked hypotheses with suspect code refs (top-1 ≥ 0.7)
- P2: /analyze returns hypotheses + code patches; WS streams progress
- P3: `helm install` on fresh Minikube; eval report meets targets

## Key acceptance criteria (spec §12)

- `make build` and `make test` pass on clean checkout
- Post /analyze matches ground-truth function for ≥ 70% of eval cases (top-1)
- At least one returned `fix.replacement` compiles and passes the seeded test
- `/metrics` exposes all prometheus collectors from spec §4.7
- Total Claude API spend < $50 across whole thesis

## What to avoid

- **No Terraform, cloud IaC, multi-region, service mesh, vector DBs** — explicitly out of scope (spec §1.2)
- **No auto-remediation** — agent suggests fixes, does not apply them
- **No auth beyond optional static `X-API-Key` header** — spec §4.1, DEVOPS §5.1
- **No database** — in-process LRU+TTL cache only
- **Only Go target codebases** — spec §1.2 non-goals
