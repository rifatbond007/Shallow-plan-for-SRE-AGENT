# SRE AI Agent - DevOps Handbook (v2)

> **Source of truth:** [`SPECIFICATION.md`](../SPECIFICATION.md), §9.
> This document covers CI/CD, security hygiene, and operational runbooks.
> Cloud-specific material (Terraform, Istio, AWS Secrets Manager, etc.) is
> explicitly removed. If anything here contradicts the spec, the spec wins.

---

## 1. What we keep, what we cut

| Old (v1) | New (v2) | Why |
|----------|----------|-----|
| Terraform modules | removed | not a v1 deliverable |
| Istio service mesh | removed | single binary; not needed |
| AWS Secrets Manager / Vault | removed | `.env` + K8s Secret is enough locally |
| PagerDuty / multi-region DR | removed | one laptop, one person |
| Chaos engineering | removed | not part of acceptance criteria |
| External secret operators | removed | same reason |
| Multi-cluster failover | removed | out of scope |
| **CI/CD (GitHub Actions)** | **kept** | needed for `make test` on every PR |
| **Docker + docker-compose** | **kept** | primary demo path |
| **K8s manifests + Helm** | **kept** | deployable acceptance criterion |
| **Prometheus + Grafana** | **kept** | observability is a deliverable |
| **Rate limiting + non-root user + read-only FS** | **kept** | security hygiene baseline |

---

## 2. Tooling

| Category | Tool | Version | Why |
|----------|------|---------|-----|
| Language | Go | 1.22+ | required for `go/parser`/`go/ast` |
| Container | Docker | 24+ | multi-stage builds |
| Orchestration | Kubernetes | 1.28+ | via Minikube/Kind |
| Package manager | Helm | 3.14+ | one chart for all envs |
| CI/CD | GitHub Actions | n/a | free for the thesis |
| Linter | `golangci-lint` | v1.59+ | standard |
| Security scan | `gosec` | latest | static analysis for Go |
| SBOM | `trivy` | latest | container CVE scan (optional) |
| Logging | `zap` | v1.27+ | in-process |
| Metrics | `prometheus/client_golang` | v1.20+ | standard |
| Reverse proxy / dashboard | (none) | — | not needed |

---

## 3. Local development

### 3.1 Prereqs (one-time)

```bash
brew install go docker helm kubectl
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 3.2 First run

```bash
git clone <repo>
cd sre-ai-agent
cp .env.example .env
# edit .env: SRE_AGENT_ANTHROPIC_API_KEY=sk-ant-...

make run         # → http://localhost:8080/api/v1/healthz
```

### 3.3 Day-to-day

```bash
make build       # compile to ./bin/sre-agent
make test        # go test ./...
make lint        # golangci-lint run
make run         # go run ./cmd/agent
make docker-build
make docker-run  # docker compose up
make eval        # go test ./tests/eval/...
```

---

## 4. CI/CD

### 4.1 Pipeline (`.github/workflows/ci.yml`)

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go mod download
      - run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
      - run: golangci-lint run --timeout=5m

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go mod download
      - run: go test -v -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out

  build-image:
    needs: [lint, test]
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ghcr.io/${{ github.repository }}
          tags: |
            type=sha,prefix=
            type=raw,value=latest,enable={{is_default_branch}}
      - uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

### 4.2 Secrets in CI

- `SRE_AGENT_ANTHROPIC_API_KEY` is **never** in CI. The eval job (if added
  later) would be triggered manually with the key as a repo secret.
- Image push uses the auto-issued `GITHUB_TOKEN`.

### 4.3 Deployment from CI

There is **no** auto-deploy step. Deployment is local via `helm install`
or `docker compose up`. This is intentional — see SPECIFICATION.md §1.2.

---

## 5. Security baseline

### 5.1 Application

- API key auth on `/api/v1/analyze` via static `X-API-Key` header
  (set via `SRE_AGENT_API_KEY` env var; if unset, the endpoint is open
  and a warning is logged at startup).
- Rate limiting: in-process token bucket per IP. Defaults: 5 RPS, burst 10.
- Input validation: log body capped at `SRE_AGENT_MAX_LOG_BYTES` (default 5 MB).
- Output: no PII is logged; only request IDs and timings.
- Dependencies: pinned in `go.sum`; Dependabot enabled on the repo.

### 5.2 Container

- Runs as non-root (`uid=10001`).
- Read-only root filesystem.
- Drops all Linux capabilities.
- `securityContext`:
  ```yaml
  runAsNonRoot: true
  runAsUser: 10001
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  ```

### 5.3 Supply chain (lightweight)

- `gosec` runs in CI; findings fail the build on `severity: high`.
- `trivy image sre-agent:dev` is documented in `docs/DEPLOY.md` but not
  run in CI by default (it's slow and noisy on a small image).
- Image signing (Cosign) is **not** in v1 — the image is only ever
  loaded locally. Add it if you ever publish publicly.

---

## 6. Observability

### 6.1 Metrics

See SPECIFICATION.md §4.7 for the full list. Top three to watch:

- `sre_agent_analyze_duration_seconds` (histogram) — overall latency
- `sre_agent_claude_api_errors_total` (counter) — API health
- `sre_agent_active_websockets` (gauge) — streaming load

### 6.2 Dashboards

A single Grafana dashboard is provisioned by compose at startup:

- `configs/grafana/dashboards/sre-agent.json` (committed)
- `configs/grafana/provisioning/dashboards/sre-agent.yml` (provisioning)

### 6.3 Logs

- Structured JSON to stdout (`zap` with `encoding: json` in production).
- In dev (`SRE_AGENT_LOG_LEVEL=debug`), pretty console output.
- Local view: `docker compose logs -f sre-agent` or `kubectl logs -f`.

---

## 7. Operational runbooks

### 7.1 Agent won't start

```bash
# 1. Check container status
docker compose ps

# 2. Read the logs
docker compose logs --tail=200 sre-agent

# 3. Verify env
docker compose exec sre-agent env | grep SRE_AGENT_

# 4. Verify Anthropic key
docker compose exec sre-agent sh -c 'echo $SRE_AGENT_ANTHROPIC_API_KEY | head -c 12'
```

Common causes: missing `ANTHROPIC_API_KEY`, port 8080 already in use,
read-only volume not mounted.

### 7.2 Analyze requests fail with 502

Claude upstream issue. Check:

```bash
curl http://localhost:8080/metrics | grep claude_api_errors_total
```

If `claude_api_errors_total` is climbing, the Claude API is degraded.
The agent retries 3× with backoff, then returns 502.

### 7.3 Eval accuracy drops after a Claude model change

1. Check `SRE_AGENT_ANTHROPIC_MODEL` — pin a specific version.
2. Re-run `make eval`. Compare `tests/eval/report.md` to the previous report.
3. If a hypothesis prompt regresses, edit `prompts/hypothesis.txt` and re-run.

### 7.4 AST cache is stale (new bugs in sample-app not detected)

```bash
# Compose
docker compose down
docker volume rm sre-ai-agent_sre-cache
docker compose up -d

# K8s
kubectl -n sre-agent delete pod -l app=sre-agent
```

### 7.5 Reset everything

```bash
docker compose down -v
docker compose build --no-cache
docker compose up -d
```

---

## 8. Disaster recovery

There is **no** DR plan because there is **no** state. Everything is in
git. The AST cache rebuilds in seconds. The eval dataset is committed.

If the laptop dies, the time to recover on a new one is roughly:

1. `git clone` → 1 minute
2. `brew install go docker` → 5 minutes
3. `cp .env.example .env` → 30 seconds
4. `docker compose up -d` → 2 minutes
5. `make eval` → 1 minute

**Total: under 10 minutes.** That's better RTO than most production systems.

---

## 9. Implementation roadmap

Mirrors SPECIFICATION.md §7 / PLAN.md.

| Phase | Weeks | Gate |
|-------|-------|------|
| 0 | 1 | `/healthz` returns 200 |
| 1 | 2–4 | Hypotheses with code refs |
| 2 | 5–6 | Hypothesis + fix; WS streaming |
| 3 | 7–8 | K8s + Helm + docs + eval |

---

## 10. Quick reference

```bash
# Development
make build && make test && make run

# Compose
docker compose up -d
docker compose logs -f sre-agent
docker compose down -v

# Minikube / Helm
eval $(minikube docker-env)
docker build -t sre-agent:dev .
helm install sre-agent ./configs/helm/sre-agent --namespace sre-agent --create-namespace
kubectl -n sre-agent port-forward svc/sre-agent 8080:8080

# Evaluation
make eval
cat tests/eval/report.md

# Cleanup
docker compose down -v
helm uninstall sre-agent -n sre-agent
```

---

## 11. Out-of-scope (explicitly)

The following were in v1 and are removed in v2. Refer back to them only if
your supervisor explicitly asks:

- Terraform (`terraform/` directory)
- AWS-specific manifests, IAM roles, Secrets Manager
- PagerDuty, Slack alerting
- Multi-cluster failover / multi-region
- Chaos engineering (Litmus, ChaosMesh)
- Service mesh (Istio)
- External secret operators (External Secrets Operator, Vault Agent)
- Distributed tracing (OpenTelemetry, Tempo, Jaeger)
- ELK / Loki for log aggregation

If you genuinely need any of these for your defense, keep the answer short:
"Out of thesis scope; documented in SPECIFICATION.md §1.2 as future work."
