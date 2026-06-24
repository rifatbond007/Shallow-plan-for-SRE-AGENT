# SRE AI Agent - Local Infrastructure (v2)

> **Source of truth:** [`SPECIFICATION.md`](./SPECIFICATION.md), §8.
> This document covers the **local** deployment story only.
> Cloud (AWS/GCP/Azure) is explicitly out of scope — see SPECIFICATION.md §1.2.
> If anything here contradicts the spec, the spec wins.

---

## 1. Why local-only

The previous plan aimed at a production-grade cloud setup. That is the right
goal for a SRE platform team; **it is the wrong goal for a BAIUST thesis.**

This project runs end-to-end on a single laptop with Docker and (optionally)
Minikube or Kind. A reviewer can reproduce everything in 10 minutes. That
property is worth more than a Terraform module that nobody will run.

---

## 2. Components on the laptop

```
┌──────────────────────────────────────────────────────────────┐
│  Laptop                                                      │
│                                                              │
│  ┌─────────────────────────┐    ┌────────────────────────┐  │
│  │  Docker Desktop /       │    │  Minikube / Kind       │  │
│  │  docker compose         │    │  (optional)            │  │
│  │                         │    │                        │  │
│  │  ┌───────────────────┐  │    │  ┌──────────────────┐  │  │
│  │  │ sre-agent         │  │    │  │ sre-agent Pod    │  │  │
│  │  │ :8080             │  │    │  │ :8080            │  │  │
│  │  └───────────────────┘  │    │  └──────────────────┘  │  │
│  │  ┌───────────────────┐  │    │                        │  │
│  │  │ prometheus :9090  │  │    │  (Helm chart installed │  │
│  │  └───────────────────┘  │    │   into default ns)     │  │
│  │  ┌───────────────────┐  │    │                        │  │
│  │  │ grafana :3000     │  │    │                        │  │
│  │  └───────────────────┘  │    │                        │  │
│  └─────────────────────────┘    └────────────────────────┘  │
│                                                              │
│  Volumes:                                                    │
│   • ./tests/data/code/sample-app  →  /codebase (ro)          │
│   • sre-cache                    →  /tmp/sre-agent/cache    │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

There is no database, no Redis, no S3, no NAT gateway, no VPC. The agent
process is the system.

---

## 3. Container

### 3.1 Image

- **Base build:** `golang:1.22-alpine`
- **Base run:** `alpine:3.20`
- **Multi-stage build** with CGO disabled, stripped binary
- Image size target: **< 50 MB**
- Runs as non-root user (`appuser`, uid 10001)
- Read-only root filesystem; writable `/tmp` and `/cache`
- Healthcheck via `wget -q -O /dev/null http://localhost:8080/api/v1/healthz` (no curl needed → smaller image)

### 3.2 Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/sre-agent ./cmd/agent

# Run stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget tzdata && \
    adduser -D -u 10001 -g '' appuser

WORKDIR /app
COPY --from=builder /out/sre-agent /usr/local/bin/sre-agent
COPY prompts /app/prompts
COPY configs/prometheus /app/configs/prometheus

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/api/v1/healthz || exit 1

ENTRYPOINT ["sre-agent"]
```

---

## 4. docker-compose (the primary demo path)

### 4.1 `configs/docker-compose.yml`

```yaml
services:
  sre-agent:
    build:
      context: ..
      dockerfile: Dockerfile
    image: sre-agent:dev
    container_name: sre-agent
    restart: unless-stopped
    ports:
      - "8080:8080"
    env_file:
      - ../.env
    environment:
      SRE_AGENT_LOG_LEVEL: info
      SRE_AGENT_CODEBASE_PATH: /codebase
      SRE_AGENT_CODEBASE_CACHE_DIR: /cache
    volumes:
      - ../tests/data/code/sample-app:/codebase:ro
      - sre-cache:/cache
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://localhost:8080/api/v1/healthz"]
      interval: 15s
      timeout: 5s
      retries: 5
    networks: [sre]

  prometheus:
    image: prom/prometheus:v2.55.0
    container_name: sre-prom
    ports: ["9090:9090"]
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
    networks: [sre]

  grafana:
    image: grafana/grafana:11.2.0
    container_name: sre-grafana
    ports: ["3000:3000"]
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
      GF_AUTH_ANONYMOUS_ENABLED: "true"
      GF_AUTH_ANONYMOUS_ORG_ROLE: Viewer
    volumes:
      - ./grafana/dashboards:/var/lib/grafana/dashboards:ro
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
    depends_on: [prometheus]
    networks: [sre]

volumes:
  sre-cache:

networks:
  sre:
    driver: bridge
```

### 4.2 Bring-up

```bash
# From repo root
cp sre-ai-agent/.env.example sre-ai-agent/.env
# Edit .env: set SRE_AGENT_ANTHROPIC_API_KEY

cd sre-ai-agent/configs
docker compose up -d

# Wait for healthy
docker compose ps
curl -fsS http://localhost:8080/api/v1/healthz
```

---

## 5. Kubernetes (Minikube / Kind) — optional

> The thesis demo uses docker-compose. This section is here for the
> "deployable" requirement in the acceptance criteria.

### 5.1 Prereqs

```bash
# One of:
brew install minikube
brew install kind

# kubectl + helm
brew install kubectl helm
```

### 5.2 Plain manifests (`configs/k8s/`)

```
configs/k8s/
├── namespace.yaml
├── configmap.yaml
├── secret.yaml.example      # copy to secret.yaml and fill in
├── deployment.yaml
├── service.yaml
└── ingress.yaml             # optional, only if nginx-ingress is installed
```

`deployment.yaml` (sketch):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sre-agent
  namespace: sre-agent
spec:
  replicas: 2
  selector:
    matchLabels: { app: sre-agent }
  template:
    metadata:
      labels: { app: sre-agent }
    spec:
      containers:
        - name: sre-agent
          image: sre-agent:dev          # use `eval $(minikube docker-env)` + `docker build`
          imagePullPolicy: Never
          ports: [{ containerPort: 8080, name: http }]
          envFrom:
            - configMapRef: { name: sre-agent-config }
            - secretRef:    { name: sre-agent-secrets }
          readinessProbe:
            httpGet: { path: /api/v1/readyz, port: 8080 }
            initialDelaySeconds: 5
            periodSeconds: 5
          livenessProbe:
            httpGet: { path: /api/v1/healthz, port: 8080 }
            initialDelaySeconds: 15
            periodSeconds: 15
          resources:
            requests: { cpu: "200m", memory: "256Mi" }
            limits:   { cpu: "1",    memory: "512Mi" }
          volumeMounts:
            - { name: cache, mountPath: /cache }
            - { name: codebase, mountPath: /codebase, readOnly: true }
      volumes:
        - name: cache
          emptyDir: {}
        - name: codebase
          hostPath:
            path: /path/to/sample-app    # adjust for your laptop
```

### 5.3 Helm chart (`configs/helm/sre-agent/`)

Standard Helm v3 chart with:

- `Chart.yaml` (apiVersion v2)
- `values.yaml` (defaults)
- `values-dev.yaml` (dev overrides: 1 replica, verbose logs, no resource limits)
- `templates/deployment.yaml`
- `templates/service.yaml`
- `templates/configmap.yaml`
- `templates/secret.yaml`
- `templates/ingress.yaml`

### 5.4 Bring-up with Minikube

```bash
# Build image inside Minikube's Docker daemon
eval $(minikube docker-env)
cd sre-ai-agent
docker build -t sre-agent:dev .

# Install
helm install sre-agent ./configs/helm/sre-agent \
    --namespace sre-agent --create-namespace

# Verify
kubectl -n sre-agent get pods
kubectl -n sre-agent port-forward svc/sre-agent 8080:8080
curl http://localhost:8080/api/v1/healthz
```

---

## 6. Volumes and persistence

The agent persists only one thing: the AST cache.

| Volume | Mount | Size | Lifecycle |
|--------|-------|------|-----------|
| `sre-cache` (compose) / `emptyDir` (k8s) | `/cache` | 256 MB | Rebuilt on first run; OK to lose |

The codebase is mounted read-only. Logs are passed in the request, not stored.

---

## 7. Networking

- **Compose:** all services on a private bridge network `sre`. Only
  `sre-agent`, `prometheus`, and `grafana` are published to `localhost`.
- **Minikube/Kind:** `ClusterIP` service; access via `kubectl port-forward`
  or via the optional Ingress.
- **Outbound:** the agent needs HTTPS to `api.anthropic.com`. Both compose
  and K8s default-allow egress.

There is **no** public ingress in the default config. If you need one,
add `configs/k8s/ingress.yaml` with `nginx.ingress.kubernetes.io/...`
annotations and a TLS secret pointing to a local cert (e.g. mkcert).

---

## 8. Observability (local)

### 8.1 Prometheus scrape

`configs/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 10s

scrape_configs:
  - job_name: sre-agent
    static_configs:
      - targets: ["sre-agent:8080"]
    metrics_path: /metrics
```

### 8.2 Grafana

A single dashboard `sre-agent.json` with four panels:

1. Request rate (`rate(sre_agent_http_requests_total[5m])`)
2. Latency p50/p95/p99 (`histogram_quantile(...)`)
3. Hypotheses generated per minute
4. Claude API error rate

Provision via `configs/grafana/provisioning/datasources/prometheus.yml`
and `configs/grafana/provisioning/dashboards/sre-agent.yml`.

### 8.3 Logs

The agent writes structured JSON to stdout (`zap`). `docker compose logs -f
sre-agent` is the demo path. No log aggregation stack — that would be
overkill for one container.

---

## 9. Backup, DR, RTO/RPO

**Not applicable.** Everything is reproducible from `git clone`. There is
no state worth backing up. The AST cache rebuilds in seconds. The eval
dataset is committed.

---

## 10. Cost

**$0/month** for infrastructure (everything runs on the laptop).

The only cost is the Claude API — see `COST.md`. Target total spend across
the entire thesis development: **< $50 USD**.

---

## 11. What this document does NOT cover

- AWS/GCP/Azure deployment — out of scope, see SPECIFICATION.md §1.2
- Terraform modules — out of scope
- Multi-cluster, multi-region — out of scope
- Service mesh — out of scope
- Production secrets management (use Kubernetes Secrets or a `.env` file)
- TLS certificates for public deployment — use `mkcert` locally; Let's Encrypt
  is overkill for a thesis demo

If any of the above is required by your supervisor, treat it as a "future
work" item, not a v1 deliverable.
