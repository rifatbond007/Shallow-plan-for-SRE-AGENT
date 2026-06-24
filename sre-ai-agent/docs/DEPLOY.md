# Deployment Guide

## Prerequisites

- Go 1.24+
- Docker & Docker Compose (local)
- Kubernetes cluster + Helm 3+ (production)

---

## Local — Docker Compose

```bash
# Build and start
docker compose -f configs/docker-compose.yml up --build -d

# Check logs
docker compose -f configs/docker-compose.yml logs -f

# Stop
docker compose -f configs/docker-compose.yml down
```

Mounts `tests/data/code/sample-app` as `/codebase` inside the container. Set your LLM API key in `.env` before starting.

---

## Local — Bare Metal

```bash
# Build
make build

# Run with .env
make run

# Or set vars directly
SRE_AGENT_LLM_PROVIDER=claude \
SRE_AGENT_ANTHROPIC_API_KEY=sk-ant-... \
./bin/sre-agent
```

---

## Production — Kubernetes with Helm

### 1. Build and push the Docker image

```bash
docker build -t your-registry/sre-agent:latest .
docker push your-registry/sre-agent:latest
```

### 2. Install the Helm chart

```bash
# With inline secrets (dev)
helm upgrade --install sre-agent ./deploy/charts/sre-agent \
  --set image.repository=your-registry/sre-agent \
  --set image.tag=latest \
  --set secrets.geminiApiKey="your-gemini-key"

# Or with a values file
cat > values-prod.yaml <<EOF
image:
  repository: your-registry/sre-agent
  tag: v1.0.0
config:
  logLevel: info
  llmProvider: gemini
secrets:
  geminiApiKey: "your-gemini-key"
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
resources:
  limits:
    cpu: "1"
    memory: 512Mi
  requests:
    cpu: 250m
    memory: 128Mi
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: sre-agent.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - hosts:
        - sre-agent.example.com
      secretName: sre-agent-tls
EOF

helm upgrade --install sre-agent ./deploy/charts/sre-agent -f values-prod.yaml
```

### 3. Codebase volume

The agent needs access to your Go service's source code. Options:

- **PVC:** Pre-populate a PersistentVolumeClaim with the codebase
- **HostPath:** Mount a node directory (development only)
- **Init container:** Git clone into an emptyDir

```yaml
codebase:
  enabled: true
  existingClaim: "my-codebase-pvc"
```

### 4. Verify

```bash
kubectl get pods -l app.kubernetes.io/name=sre-agent
kubectl port-forward deployment/sre-agent 8080:8080
curl http://localhost:8080/api/v1/healthz
```

---

## Configuration

All configuration is via environment variables (see `docs/API.md` for the full reference). In Helm, these are split between:

- **ConfigMap:** Non-sensitive settings (port, limits, timeouts)
- **Secret:** API keys (Anthropic, Gemini)

---

## Resource Requirements

| Environment | CPU | Memory | Storage |
|---|---|---|---|
| Dev / eval | 0.5 core | 256 MiB | 100 MiB (codebase) |
| Production | 1 core | 512 MiB | 1 GiB (codebase + cache) |

---

## Monitoring

Prometheus metrics at `/metrics`. A Prometheus scrape config is included in `configs/prometheus.yml`.
