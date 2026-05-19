# DevOps Operations Handbook

> **SRE AI Agent Project** - DevOps Implementation Guidelines
> 
> This document outlines the DevOps practices, tools, and workflows for implementing the SRE AI Agent project.

---

## Table of Contents

1. [Tech Stack Recommendation](#1-tech-stack-recommendation)
2. [Development Environment](#2-development-environment)
3. [Containerization](#3-containerization)
4. [Kubernetes Deployment](#4-kubernetes-deployment)
5. [Helm Charts](#5-helm-charts)
6. [Infrastructure as Code](#6-infrastructure-as-code)
7. [CI/CD Pipeline](#7-cicd-pipeline)
8. [Monitoring & Observability](#8-monitoring--observability)
9. [Security Practices](#9-security-practices)
10. [Disaster Recovery](#10-disaster-recovery)
11. [Implementation Roadmap](#11-implementation-roadmap)

---

## 1. Tech Stack Recommendation

### Core Technologies

| Category | Tool | Version | Purpose |
|----------|------|---------|---------|
| **Language** | Go | 1.21+ | Primary application language |
| **Container** | Docker | 24+ | Container runtime |
| **Orchestration** | Kubernetes | 1.28+ | Container orchestration |
| **Package Manager** | Helm | 3.14+ | Kubernetes package management |
| **IaC** | Terraform | 1.6+ | Infrastructure provisioning |
| **CI/CD** | GitHub Actions | Latest | Continuous integration/deployment |
| **Service Mesh** | Istio | 1.20+ | Traffic management (optional) |

### Supporting Tools

| Category | Tool | Purpose |
|----------|------|---------|
| **Logging** | Fluent Bit + Loki | Log aggregation |
| **Metrics** | Prometheus | Metrics collection |
| **Visualization** | Grafana | Dashboards |
| **Alerting** | Alertmanager + PagerDuty | Alert management |
| **Secrets** | HashiCorp Vault | Secrets management |
| **Tracing** | Tempo | Distributed tracing |
| **Container Registry** | GitHub Container Registry | Image storage |

---

## 2. Development Environment

### Required Tools Installation

```bash
# macOS (using Homebrew)
brew install go docker kubernetes-cli helm terraform tfenv kubectx

# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y golang-go docker.io kubectl helm terraform

# Verify installations
go version
docker --version
kubectl version --client
helm version
terraform --version
```

### Local Development Setup

```bash
# Clone repository
git clone https://github.com/rifatbond007/Shallow-plan-for-SRE-AGENT.git
cd Shallow-plan-for-SRE-AGENT/sre-ai-agent

# Install Go dependencies
go mod download
go mod tidy

# Build the application
make build

# Run locally with Docker Compose
docker-compose up -d

# Access services
# - API: http://localhost:8080
# - Swagger: http://localhost:8080/swagger
```

### Development Workflow

```
1. Create feature branch
   git checkout -b feature/your-feature

2. Make changes and test locally
   make test
   make run

3. Commit with conventional commits
   git commit -m "feat: add new feature"

4. Push and create PR
   git push origin feature/your-feature
```

---

## 3. Containerization

### Dockerfile (Multi-stage Build)

Create `Dockerfile`:

```dockerfile
# ====== BUILD STAGE ======
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -o /sre-agent ./cmd/agent

# ====== RUNNER STAGE ======
FROM alpine:3.19

WORKDIR /app

# Install CA certificates and healthcheck tools
RUN apk add --no-cache ca-certificates curl

# Copy binary from builder
COPY --from=builder /sre-agent /usr/local/bin/
COPY --from=builder /app/configs/ ./configs/

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Run application
ENTRYPOINT ["sre-agent"]
```

### Docker Compose (Local Dev)

Create `docker-compose.yml`:

```yaml
version: '3.9'

services:
  sre-agent:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - LOG_LEVEL=debug
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - REDIS_URL=redis:6379
      - DATABASE_URL=postgres://postgres:postgres@db:5432/sre_agent?sslmode=disable
    depends_on:
      redis:
        condition: service_healthy
      db:
        condition: service_healthy
    networks:
      - sre-network

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - sre-network

  db:
    image: postgres:16-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
      - POSTGRES_DB=sre_agent
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - sre-network

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./configs/prometheus.yml:/etc/prometheus/prometheus.yml
    networks:
      - sre-network

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    networks:
      - sre-network

networks:
  sre-network:
    driver: bridge

volumes:
  postgres-data:
```

---

## 4. Kubernetes Deployment

### Kubernetes Manifests Structure

```
configs/k8s/
├── namespace.yaml          # Namespace definition
├── configmap.yaml         # Application configuration
├── secret.yaml            # Sensitive data (use external secrets)
├── deployment.yaml        # Application deployment
├── service.yaml           # Service definition
├── hpa.yaml               # Horizontal pod autoscaler
├── pdb.yaml               # Pod disruption budget
├── ingress.yaml           # Ingress configuration
└── serviceaccount.yaml   # Service account
```

### Deployment Example

```yaml
# configs/k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sre-agent
  namespace: sre-agent
  labels:
    app: sre-agent
spec:
  replicas: 3
  selector:
    matchLabels:
      app: sre-agent
  template:
    metadata:
      labels:
        app: sre-agent
    spec:
      serviceAccountName: sre-agent
      containers:
        - name: sre-agent
          image: ghcr.io/rifatbond007/sre-agent:latest
          imagePullPolicy: Always
          ports:
            - containerPort: 8080
              name: http
          env:
            - name: LOG_LEVEL
              valueFrom:
                configMapKeyRef:
                  name: sre-agent-config
                  key: log_level
            - name: ANTHROPIC_API_KEY
              valueFrom:
                secretKeyRef:
                  name: sre-agent-secrets
                  key: anthropic-api-key
          resources:
            requests:
              memory: "512Mi"
              cpu: "250m"
            limits:
              memory: "1Gi"
              cpu: "1000m"
          livenessProbe:
            httpGet:
              path: /api/v1/health
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /api/v1/health/ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
```

### Service & Ingress

```yaml
# configs/k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: sre-agent
  namespace: sre-agent
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 8080
      protocol: TCP
      name: http
  selector:
    app: sre-agent
```

```yaml
# configs/k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: sre-agent
  namespace: sre-agent
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
    - host: sre-agent.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: sre-agent
                port:
                  number: 80
  tls:
    - hosts:
        - sre-agent.example.com
      secretName: sre-agent-tls
```

---

## 5. Helm Charts

### Helm Chart Structure

```
charts/sre-agent/
├── Chart.yaml
├── values.yaml
├── values.prod.yaml
├── values.staging.yaml
├── values.dev.yaml
└── templates/
    ├── _helpers.tpl
    ├── deployment.yaml
    ├── service.yaml
    ├── ingress.yaml
    ├── configmap.yaml
    ├── secret.yaml
    ├── hpa.yaml
    ├── serviceaccount.yaml
    └── tests/
        └── test-connection.yaml
```

### Chart.yaml

```yaml
apiVersion: v2
name: sre-agent
description: SRE AI Agent - AI-powered log analysis and root cause hypothesis generation
type: application
version: 1.0.0
appVersion: "1.0.0"
keywords:
  - sre
  - ai
  - logging
  - monitoring
maintainers:
  - name: rifatbond007
    url: https://github.com/rifatbond007
```

### values.yaml (Production)

```yaml
replicaCount: 3

image:
  repository: ghcr.io/rifatbond007/sre-agent
  pullPolicy: IfNotPresent
  tag: "latest"

service:
  type: ClusterIP
  port: 80
  targetPort: 8080

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
    - secretName: sre-agent-tls
      hosts:
        - sre-agent.example.com

resources:
  limits:
    cpu: 2000m
    memory: 2Gi
  requests:
    cpu: 1000m
    memory: 1Gi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  targetCPUUtilizationPercentage: 70

nodeSelector: {}

tolerations: []

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: sre-agent
          topologyKey: kubernetes.io/hostname

config:
  log_level: info
  redis_url: redis://redis:6379
  database_url: postgres://postgres:postgres@postgres:5432/sre_agent

secrets:
  anthropic_api_key: ""

prometheus:
  enabled: true
  serviceMonitor:
    enabled: true
```

### Helm Commands

```bash
# Add repository
helm repo add sre-agent https://rifatbond007.github.io/Shallow-plan-for-SRE-AGENT

# Install chart
helm install sre-agent sre-agent/sre-agent \
    --namespace sre-agent \
    --create-namespace \
    --values values.prod.yaml

# Upgrade chart
helm upgrade sre-agent sre-agent/sre-agent \
    --namespace sre-agent \
    --values values.prod.yaml

# Rollback
helm rollback sre-agent 1

# Uninstall
helm uninstall sre-agent --namespace sre-agent
```

---

## 6. Infrastructure as Code

### Terraform Structure

```
terraform/
├── main.tf                 # Main configuration
├── variables.tf            # Variable definitions
├── outputs.tf              # Output definitions
├── providers.tf            # Provider configuration
├── modules/
│   ├── vpc/               # VPC module
│   ├── eks/               # EKS cluster module
│   ├── rds/               # RDS module
│   ├── redis/             # ElastiCache module
│   └── ecr/               # ECR module
└── environments/
    ├── dev/
    ├── staging/
    └── prod/
```

### EKS Module Example

```terraform
# terraform/modules/eks/main.tf
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 19.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.28"

  vpc_id                         = module.vpc.vpc_id
  subnet_ids                     = module.vpc.private_subnets
  cluster_endpoint_public_access = true

  eks_managed_node_group_defaults = {
    ami_type       = "AL2_x86_64"
    instance_types = ["t3.medium"]
  }

  eks_managed_node_groups = {
    primary = {
      name = "primary"

      instance_types = ["t3.medium"]
      capacity_type  = "ON_DEMAND"

      min_size     = 2
      max_size     = 10
      desired_size = 3

      labels = {
        Environment = var.environment
        NodeGroup   = "primary"
      }
    }
  }

  tags = var.common_tags
}
```

---

## 7. CI/CD Pipeline

### GitHub Actions Workflow

Create `.github/workflows/ci-cd.yml`:

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  # ─────────────────────────────────────────────
  # Stage 1: Code Quality
  # ─────────────────────────────────────────────
  code-quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          args: --timeout=5m

      - name: Run gosec
        run: |
          go install github.com/securego/gosec/v2/cmd/gosec@latest
          gosec -fmt sarif -out gosec-results.sarif ./...

  # ─────────────────────────────────────────────
  # Stage 2: Test
  # ─────────────────────────────────────────────
  test:
    needs: code-quality
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out

  # ─────────────────────────────────────────────
  # Stage 3: Build & Push Image
  # ─────────────────────────────────────────────
  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=sha,prefix=
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  # ─────────────────────────────────────────────
  # Stage 4: Deploy to Kubernetes
  # ─────────────────────────────────────────────
  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    environment: production
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: us-east-1

      - name: Login to Amazon ECR
        uses: aws-actions/amazon-ecr-login@v1

      - name: Deploy to EKS
        run: |
          aws eks update-kubeconfig --name sre-agent-prod
          kubectl set image deployment/sre-agent \
            sre-agent=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }} \
            -n sre-agent

      - name: Verify deployment
        run: |
          kubectl rollout status deployment/sre-agent -n sre-agent --timeout=300s
```

### Makefile Targets

```makefile
# Makefile
.PHONY: build test run docker-build docker-run k8s-deploy k8s-delete

# Build the application
build:
	go build -o bin/sre-agent ./cmd/agent

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run locally
run:
	go run ./cmd/agent

# Build Docker image
docker-build:
	docker build -t sre-agent:latest .

# Run with Docker Compose
docker-run:
	docker-compose up -d

# Deploy to Kubernetes
k8s-deploy:
	kubectl apply -f configs/k8s/

# Delete from Kubernetes
k8s-delete:
	kubectl delete -f configs/k8s/

# Helm install
helm-install:
	helm install sre-agent ./charts/sre-agent \
		--namespace sre-agent \
		--create-namespace

# Helm upgrade
helm-upgrade:
	helm upgrade sre-agent ./charts/sre-agent \
		--namespace sre-agent

# Lint
lint:
	golangci-lint run
```

---

## 8. Monitoring & Observability

### Prometheus Configuration

```yaml
# configs/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'sre-agent'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
```

### Key Metrics to Expose

| Metric | Type | Description |
|--------|------|-------------|
| `sre_agent_requests_total` | Counter | Total HTTP requests |
| `sre_agent_request_duration_seconds` | Histogram | Request latency |
| `sre_agent_analyze_total` | Counter | Analysis operations |
| `sre_agent_hypothesis_generated_total` | Counter | Hypotheses created |
| `sre_agent_claude_api_calls_total` | Counter | Claude API calls |
| `sre_agent_claude_api_errors_total` | Counter | Claude API errors |
| `sre_agent_active_websockets` | Gauge | Active WS connections |

---

## 9. Security Practices

### Security Checklist

- [ ] Use secrets management (Vault/AWS Secrets Manager)
- [ ] Enable RBAC in Kubernetes
- [ ] Use network policies to restrict pod communication
- [ ] Enable container image scanning in CI/CD
- [ ] Run containers as non-root user
- [ ] Use read-only root filesystem
- [ ] Enable TLS/SSL for all external communications
- [ ] Implement rate limiting
- [ ] Regular security updates and patches

### Container Security

```dockerfile
# Security best practices in Dockerfile
FROM alpine:3.19 AS builder

# Build as non-root in build stage
RUN adduser -D -g '' builder
USER builder

FROM alpine:3.19

# Don't run as root
RUN adduser -D -g '' appuser
USER appuser

# Read-only filesystem
RUN rm -rf /tmp /var/tmp
VOLUME /tmp /var/tmp

# No shell access
RUN rm -f /bin/sh /bin/bash
```

---

## 10. Disaster Recovery

### Backup Strategy

| Component | Method | Frequency | Retention |
|-----------|--------|-----------|-----------|
| PostgreSQL | RDS Automated Backups | Daily | 30 days |
| Redis | AOF + RDB | Every 5 min | 7 days |
| Config | Git | On change | Forever |
| Docker Images | ECR | Every push | 90 days |

### Recovery Runbook

```bash
# 1. Check pod status
kubectl get pods -n sre-agent

# 2. Describe failing pod
kubectl describe pod <pod-name> -n sre-agent

# 3. View logs
kubectl logs <pod-name> -n sre-agent --previous

# 4. Rollback to previous version
kubectl rollout undo deployment/sre-agent -n sre-agent

# 5. Scale up if needed
kubectl scale deployment sre-agent --replicas=5 -n sre-agent
```

---

## 11. Implementation Roadmap

### Phase 1: Development Setup (Week 1)
- [ ] Set up development environment
- [ ] Configure Docker and Docker Compose
- [ ] Create basic application structure
- [ ] Set up GitHub Actions CI pipeline

### Phase 2: Containerization (Week 2)
- [ ] Create optimized Dockerfile
- [ ] Configure Docker Compose for local dev
- [ ] Set up local PostgreSQL and Redis
- [ ] Implement health checks

### Phase 3: Kubernetes (Week 3)
- [ ] Create Kubernetes manifests
- [ ] Set up namespace and RBAC
- [ ] Configure HPA and PDB
- [ ] Implement ingress with TLS

### Phase 4: Helm Charts (Week 4)
- [ ] Create Helm chart structure
- [ ] Implement values for all environments
- [ ] Add testing and linting
- [ ] Set up chart repository

### Phase 5: Infrastructure (Week 5)
- [ ] Set up Terraform for AWS
- [ ] Create EKS cluster
- [ ] Configure RDS and ElastiCache
- [ ] Set up CI/CD deployment

### Phase 6: Observability (Week 6)
- [ ] Configure Prometheus
- [ ] Create Grafana dashboards
- [ ] Set up alerting
- [ ] Implement logging pipeline

### Phase 7: Security & Compliance (Week 7)
- [ ] Implement secrets management
- [ ] Add security scanning
- [ ] Configure network policies
- [ ] Security audit

### Phase 8: Production Ready (Week 8)
- [ ] Load testing
- [ ] Chaos engineering
- [ ] Documentation
- [ ] Runbook creation

---

## Quick Reference Commands

```bash
# Development
make build          # Build application
make run            # Run locally
make test           # Run tests
make docker-build   # Build Docker image

# Kubernetes
kubectl apply -f configs/k8s/           # Deploy
kubectl get pods -n sre-agent           # Check pods
kubectl logs -f <pod> -n sre-agent      # View logs
kubectl exec -it <pod> -n sre-agent sh  # Shell access

# Helm
helm install sre-agent ./charts/sre-agent     # Install
helm upgrade sre-agent ./charts/sre-agent     # Upgrade
helm rollback sre-agent 1                      # Rollback

# Terraform
terraform init                                  # Initialize
terraform plan                                  # Plan changes
terraform apply                                 # Apply changes
terraform destroy                               # Destroy

# Monitoring
kubectl port-forward -n monitoring svc/prometheus 9090:9090
kubectl port-forward -n monitoring svc/grafana 3000:3000
```

---

## Resources

- [Go Documentation](https://go.dev/doc/)
- [Docker Documentation](https://docs.docker.com/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Helm Documentation](https://helm.sh/docs/)
- [Terraform Documentation](https://www.terraform.io/docs/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)

---

> **Last Updated:** 2026-05-19
> **Version:** 1.0.0
> **Maintainer:** rifatbond007