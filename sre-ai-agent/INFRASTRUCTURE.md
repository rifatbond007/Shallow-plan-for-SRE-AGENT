# SRE AI Agent - Infrastructure Architecture

> **Platform Lead Engineer Note**: This document outlines the infrastructure design for deploying and operating the SRE AI Agent at scale. It covers cloud architecture, networking, CI/CD, monitoring, security, and disaster recovery.

---

## 1. Infrastructure Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    CLOUD INFRASTRUCTURE                                      │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────────────────┐   │
│  │                                USER TRAFFIC LAYER                                    │   │
│  │                                                                                      │   │
│  │    ┌─────────────┐      ┌─────────────┐      ┌─────────────┐                       │   │
│  │    │   AWS/      │      │    Cloud    │      │    WAF      │                       │   │
│  │    │   Route 53  │─────▶│    Front    │─────▶│  (Firewall) │                       │   │
│  │    │  (DNS)       │      │    (ELB)    │      │             │                       │   │
│  │    └─────────────┘      └─────────────┘      └──────┬──────┘                       │   │
│  │                                                      │                               │   │
│  └──────────────────────────────────────────────────────┼───────────────────────────────┘   │
│                                                         │                                    │
│                                                         ▼                                    │
│  ┌─────────────────────────────────────────────────────────────────────────────────────┐   │
│  │                             KUBERNETES CLUSTER (EKS/GKE/AKS)                         │   │
│  │                                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────────────────────┐   │   │
│  │  │                        Namespace: sre-agent                                   │   │   │
│  │  │                                                                              │   │   │
│  │  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐            │   │   │
│  │  │  │   Pod 1    │  │   Pod 2    │  │   Pod 3    │  │   Pod N    │            │   │   │
│  │  │  │ sre-agent  │  │ sre-agent  │  │ sre-agent  │  │ sre-agent  │            │   │   │
│  │  │  │  :8080     │  │  :8080     │  │  :8080     │  │  :8080     │            │   │   │
│  │  │  └────────────┘  └────────────┘  └────────────┘  └────────────┘            │   │   │
│  │  │       │               │               │               │                     │   │   │
│  │  │       └───────────────┴───────────────┴───────────────┘                     │   │   │
│  │  │                               │                                             │   │   │
│  │  │                               ▼                                             │   │   │
│  │  │                    ┌─────────────────────┐                                 │   │   │
│  │  │                    │  Service (ClusterIP)│                                 │   │   │
│  │  │                    │     :8080           │                                 │   │   │
│  │  │                    └─────────────────────┘                                 │   │   │
│  │  └─────────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────────────────────┘   │
│                                         │                                                   │
│                                         │                                                   │
│  ┌──────────────────────────────────────┼───────────────────────────────────────────────┐   │
│  │                          DATA LAYER   │                                               │   │
│  │                                                                                      │   │
│  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐                │   │
│  │  │    PostgreSQL   │    │      Redis      │    │    S3/MinIO    │                │   │
│  │  │   (RDS/Aurora)  │    │   (ElastiCache) │    │   (Artifacts)  │                │   │
│  │  │     :5432       │    │      :6379      │    │                 │                │   │
│  │  └─────────────────┘    └─────────────────┘    └─────────────────┘                │   │
│  │                                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────────────────┐   │
│  │                              EXTERNAL SERVICES                                        │   │
│  │                                                                                      │   │
│  │  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐                │   │
│  │  │   Anthropic     │    │   Prometheus    │    │   Grafana       │                │   │
│  │  │   Claude API    │    │   (Metrics)     │    │   (Dashboards)  │                │   │
│  │  └─────────────────┘    └─────────────────┘    └─────────────────┘                │   │
│  │                                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                              │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Cloud Provider Configuration

### 2.1 AWS (Recommended)

| Resource | Service | Configuration |
|----------|---------|---------------|
| Compute | EKS | 3 nodes, t3.medium, auto-scaling |
| Database | RDS PostgreSQL | db.t3.medium, multi-AZ |
| Cache | ElastiCache Redis | cache.t3.medium |
| Storage | S3 | Standard tier, lifecycle policies |
| DNS | Route 53 | Health checks enabled |
| Load Balancer | ALB | Application Load Balancer |
| WAF | AWS WAF | Rate limiting, IP blocking |
| Secrets | Secrets Manager | Auto-rotation enabled |
| Monitoring | CloudWatch | Logs, metrics, alarms |

### 2.2 Kubernetes Cluster Specification

```yaml
# EKS Cluster Configuration
cluster:
  name: sre-agent-prod
  version: 1.28
  region: us-east-1

  # Node Groups
  nodeGroups:
    - name: sre-agent-nodes
      instanceType: t3.medium
      minSize: 2
      maxSize: 10
      desiredCapacity: 3
      volumeSize: 50
      volumeType: gp3

  # Add-ons
  addons:
    - name: vpc-cni
    - name: coredns
    - name: kube-proxy
    - name: aws-ebs-csi-driver
    - name: metrics-server
    - name: cluster-autoscaler

# Network Configuration
networking:
  vpc:
    cidr: 10.0.0.0/16
    azs: [us-east-1a, us-east-1b, us-east-1c]
    privateSubnets:
      - 10.0.1.0/24
      - 10.0.2.0/24
      - 10.0.3.0/24
    publicSubnets:
      - 10.0.101.0/24
      - 10.0.102.0/24
      - 10.0.103.0/24
```

---

## 3. Network Architecture

### 3.1 VPC Design

```
┌─────────────────────────────────────────────────────────────────┐
│                         VPC (10.0.0.0/16)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  PUBLIC SUBNETS                          │   │
│  │         10.0.101.0/24  |  10.0.102.0/24  | 10.0.103.0/24 │   │
│  │                                                          │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │   NAT GW 1   │  │   NAT GW 2   │  │   NAT GW 3   │  │   │
│  │  │ (us-east-1a) │  │ (us-east-1b) │  │ (us-east-1c) │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                 PRIVATE SUBNETS (WORKER NODES)          │   │
│  │         10.0.1.0/24   |  10.0.2.0/24   |  10.0.3.0/24   │   │
│  │                                                          │   │
│  │  ┌──────────────────────────────────────────────────┐  │   │
│  │  │              EKS CLUSTER                          │  │   │
│  │  │   ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐   │  │   │
│  │  │   │ Pod 1  │ │ Pod 2  │ │ Pod 3  │ │ Pod N  │   │  │   │
│  │  │   └────────┘ └────────┘ └────────┘ └────────┘   │  │   │
│  │  └──────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                 DATABASE SUBNETS                         │   │
│  │         10.0.11.0/24 |  10.0.12.0/24 |  10.0.13.0/24    │   │
│  │                                                          │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │  PostgreSQL  │  │    Redis     │  │   MinIO      │  │   │
│  │  │   Primary    │  │   Cluster    │  │   (Local)    │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Security Groups

| Security Group | Inbound | Outbound | Purpose |
|----------------|---------|----------|---------|
| `sg-eks-nodes` | All from VPC | All | EKS worker nodes |
| `sg-sre-agent` | 8080 from SG-EKS-NODES | All | SRE Agent pods |
| `sg-postgres` | 5432 from SG-SRE-AGENT | - | PostgreSQL |
| `sg-redis` | 6379 from SG-SRE-AGENT | - | Redis |
| `sg-alb` | 80, 443 from Anywhere | 8080 to SG-SRE-AGENT | Load Balancer |

---

## 4. CI/CD Pipeline

### 4.1 Pipeline Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    CI/CD PIPELINE                                            │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                              │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐                  │
│  │  Code   │───▶│  Build  │───▶│  Test   │───▶│  Build  │───▶│ Deploy  │                  │
│  │  Push   │    │         │    │         │    │  Docker │    │ to K8s  │                  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘    └────┬────┘                  │
│                                                                       │                      │
│  ┌────────────────────────────────────────────────────────────────────┼──────────────────┐ │
│  │                            GITHUB ACTIONS                          │                  │ │
│  │                                                                     │                  │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │                  │ │
│  │  │   Checkout   │  │   Lint &     │  │  Docker      │            │                  │ │
│  │  │   Code       │  │   Test       │  │  Build & Push│            │                  │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘            │                  │ │
│  │                                                                     │                  │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │                  │ │
│  │  │  Security    │  │   K8s        │  │  Smoke       │            │                  │ │
│  │  │   Scan       │  │   Deploy     │  │   Tests      │            │                  │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘            │                  │ │
│  └────────────────────────────────────────────────────────────────────┴──────────────────┘ │
│                                                                                              │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 GitHub Actions Workflow

```yaml
# .github/workflows/ci-cd.yml
name: CI/CD Pipeline

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  # ─────────────────────────────────────────────────────────────
  # Stage 1: Code Quality
  # ─────────────────────────────────────────────────────────────
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

      - name: Upload security results
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: gosec-results.sarif

  # ─────────────────────────────────────────────────────────────
  # Stage 2: Test
  # ─────────────────────────────────────────────────────────────
  test:
    needs: code-quality
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Download dependencies
        run: go mod download

      - name: Run unit tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out

  # ─────────────────────────────────────────────────────────────
  # Stage 3: Build and Push Docker Image
  # ─────────────────────────────────────────────────────────────
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

  # ─────────────────────────────────────────────────────────────
  # Stage 4: Deploy to Kubernetes
  # ─────────────────────────────────────────────────────────────
  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    environment: production
    permissions:
      contents: read
      id-token: write
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: us-east-1

      - name: Login to Amazon ECR
        id: login-ecr
        uses: aws-actions/amazon-ecr-login@v1

      - name: Deploy to EKS
        uses: kubectl/kubectl-setup@v1
        with:
          kubeconfig: ${{ secrets.KUBECONFIG }}

      - name: Update deployment image
        run: |
          export KUBECONFIG=${{ secrets.KUBECONFIG }}
          kubectl set image deployment/sre-agent \
            sre-agent=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }} \
            -n sre-agent

      - name: Verify deployment
        run: |
          export KUBECONFIG=${{ secrets.KUBECONFIG }}
          kubectl rollout status deployment/sre-agent -n sre-agent --timeout=300s

      - name: Run smoke tests
        run: |
          export KUBECONFIG=${{ secrets.KUBECONFIG }}
          kubectl run smoke-test --image=curlimages/curl --restart=Never \
            -- curl -f http://sre-agent:8080/api/v1/health
          kubectl wait --for=condition=complete job/smoke-test --timeout=60s
```

### 4.3 Deployment Environments

| Environment | Cluster | Replicas | Resources | Auto-scaling |
|-------------|---------|----------|-----------|--------------|
| Development | dev-cluster | 1 | 512Mi, 0.5 CPU | No |
| Staging | staging-cluster | 2 | 1Gi, 1 CPU | Yes (2-5) |
| Production | prod-cluster | 3-10 | 2Gi, 2 CPU | Yes (3-20) |

---

## 5. Monitoring & Observability

### 5.1 Monitoring Stack

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              OBSERVABILITY STACK                                            │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                              │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐                        │
│  │   Prometheus    │◀───│   SRE Agent     │───▶│    Grafana      │                        │
│  │   (Scrape)      │    │  (Metrics)      │    │   (Dashboard)   │                        │
│  └────────┬────────┘    └─────────────────┘    └────────┬────────┘                        │
│           │                                              │                                 │
│           │    ┌─────────────────┐                       │                                 │
│           ├───▶│   AlertManager  │◀──────────────────────┘                                 │
│           │    └────────┬────────┘                                                         │
│           │             │                                                                  │
│           │             ▼                                                                  │
│           │    ┌─────────────────┐                                                         │
│           └───▶│    PagerDuty    │                                                         │
│                │   (On-Call)     │                                                         │
│                └─────────────────┘                                                         │
│                                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                              LOGGING                                                 │    │
│  │                                                                                      │    │
│  │  SRE Agent ──▶ Fluent Bit ──▶ Loki ──▶ Grafana                                    │    │
│  │                                                                                      │    │
│  └────────────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                              TRACING                                                 │    │
│  │                                                                                      │    │
│  │  SRE Agent ──▶ Tempo ──▶ Grafana                                                   │    │
│  │                                                                                      │    │
│  └────────────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                              │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 5.2 Key Metrics

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| `sre_agent_requests_total` | Counter | Total API requests | N/A |
| `sre_agent_request_duration_seconds` | Histogram | Request latency | p99 > 5s |
| `sre_agent_analyze_total` | Counter | Analysis requests | N/A |
| `sre_agent_hypothesis_generated_total` | Counter | Hypotheses generated | N/A |
| `sre_agent_claude_api_calls_total` | Counter | Claude API calls | N/A |
| `sre_agent_claude_api_errors_total` | Counter | Claude API errors | > 5% |
| `sre_agent_active_websockets` | Gauge | Active WS connections | > 1000 |
| `sre_agent_codebase_parse_duration` | Histogram | Code parsing time | p95 > 30s |

### 5.3 Grafana Dashboards

```json
{
  "dashboard": {
    "title": "SRE AI Agent Overview",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {"expr": "rate(sre_agent_requests_total[5m])"}
        ]
      },
      {
        "title": "Latency (p50, p95, p99)",
        "targets": [
          {"expr": "histogram_quantile(0.50, rate(sre_agent_request_duration_seconds_bucket[5m]))"},
          {"expr": "histogram_quantile(0.95, rate(sre_agent_request_duration_seconds_bucket[5m]))"},
          {"expr": "histogram_quantile(0.99, rate(sre_agent_request_duration_seconds_bucket[5m]))"}
        ]
      },
      {
        "title": "Analysis Success Rate",
        "targets": [
          {"expr": "rate(sre_agent_hypothesis_generated_total[5m]) / rate(sre_agent_analyze_total[5m])"}
        ]
      },
      {
        "title": "Claude API Error Rate",
        "targets": [
          {"expr": "rate(sre_agent_claude_api_errors_total[5m]) / rate(sre_agent_claude_api_calls_total[5m])"}
        ]
      }
    ]
  }
}
```

---

## 6. Security Architecture

### 6.1 Security Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                      SECURITY LAYERS                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  LAYER 1: Edge Security                                  │   │
│  │  - WAF (Web Application Firewall)                       │   │
│  │  - DDoS Protection (CloudFront)                         │   │
│  │  - Rate Limiting                                        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                           │                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  LAYER 2: Network Security                               │   │
│  │  - VPC Isolation                                        │   │
│  │  - Security Groups                                      │   │
│  │  - Private Subnets                                      │   │
│  │  - TLS/SSL Termination                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                           │                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  LAYER 3: Application Security                          │   │
│  │  - API Authentication (JWT/OAuth)                       │   │
│  │  - Input Validation                                     │   │
│  │  - SQL Injection Prevention                             │   │
│  │  - XSS Protection                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                           │                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  LAYER 4: Data Security                                 │   │
│  │  - Encryption at Rest (KMS)                             │   │
│  │  - Encryption in Transit                                │   │
│  │  - Secrets Management                                   │   │
│  │  - Key Rotation                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 Secrets Management

| Secret | Storage | Rotation |
|--------|---------|----------|
| `ANTHROPIC_API_KEY` | AWS Secrets Manager | 90 days |
| `POSTGRES_PASSWORD` | AWS Secrets Manager | 30 days |
| `REDIS_PASSWORD` | AWS Secrets Manager | 30 days |
| `JWT_SECRET` | AWS Secrets Manager | 180 days |

### 6.3 IAM Roles & Policies

```yaml
# EKS Pod Execution Role
- Policy: AmazonEKSWorkerNodePolicy
- Policy: AmazonEKS_CNI_Policy
- Policy: AmazonEBSCSIDriverPolicy
- Policy: SecretsManagerReadWrite (for pod)

# SRE Agent Service Account Role
- Policy: Allow describe pods
- Policy: Allow read secrets
- Policy: Allow CloudWatch logs
```

---

## 7. Backup & Disaster Recovery

### 7.1 Backup Strategy

| Component | Backup Method | Frequency | Retention |
|-----------|---------------|-----------|-----------|
| PostgreSQL | AWS RDS Automated Backups | Daily | 30 days |
| PostgreSQL | Point-in-time Recovery | Continuous | 35 days |
| Redis | AOF + RDB | Every 5 min | 7 days |
| Config Files | Git | On change | Forever |
| Secrets | AWS Secrets Manager | N/A | Managed |
| Docker Images | ECR | Every push | 90 days |

### 7.2 Disaster Recovery Plan

```
┌─────────────────────────────────────────────────────────────────┐
│                    DISASTER RECOVERY PROCEDURE                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. DETECTION (0-5 min)                                         │
│     ├── Prometheus alerts trigger                               │
│     ├── PagerDuty notifies on-call                              │
│     └── Auto-scaling triggers                                   │
│                                                                  │
│  2. CONTAINMENT (5-15 min)                                      │
│     ├── Isolate affected pods                                   │
│     ├── Enable failover mode                                    │
│     └── Route traffic to healthy nodes                          │
│                                                                  │
│  3. RECOVERY (15-60 min)                                        │
│     ├── Restore from latest backup                              │
│     ├── Deploy new pods                                         │
│     ├── Verify health checks                                    │
│                                                                  │
│  4. POST-RECOVERY (60+ min)                                     │
│     ├── Run smoke tests                                         │
│     ├── Update stakeholders                                     │
│     └── Document incident                                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 7.3 RTO/RPO Targets

| Recovery Objective | Target | Implementation |
|--------------------|--------|----------------|
| RTO (Recovery Time Objective) | < 15 min | Multi-AZ deployment, auto-failover |
| RPO (Recovery Point Objective) | < 5 min | Redis AOF, PostgreSQL continuous backup |

---

## 8. Cost Optimization

### 8.1 Cost Breakdown (Monthly - AWS)

| Resource | Size | Monthly Cost (Estimate) |
|----------|------|------------------------|
| EKS Cluster | 3 t3.medium | $150 |
| EKS Worker Nodes | 3 t3.medium | $200 |
| RDS PostgreSQL | db.t3.medium | $150 |
| ElastiCache Redis | cache.t3.medium | $80 |
| S3 Storage | 50 GB | $5 |
| Data Transfer | 100 GB | $10 |
| CloudWatch | Basic | $50 |
| Route 53 | 1 hosted zone | $1 |
| **Total** | | **~$646/month** |

### 8.2 Optimization Recommendations

- Use spot instances for non-production environments (60-70% savings)
- Enable EKS cluster auto-scaler
- Use S3 Intelligent-Tiering for logs
- Implement request caching to reduce API calls
- Use reserved instances for production (40% savings)

---

## 9. Environment-Specific Configurations

### 9.1 Development Environment

```yaml
# dev-values.yaml
replicas: 1

resources:
  limits:
    cpu: "500m"
    memory: "512Mi"
  requests:
    cpu: "250m"
    memory: "256Mi"

autoscaling:
  enabled: false

monitoring:
  enabled: true
  sampleRate: 1.0
```

### 9.2 Production Environment

```yaml
# prod-values.yaml
replicas: 3

resources:
  limits:
    cpu: "2000m"
    memory: "2Gi"
  requests:
    cpu: "1000m"
    memory: "1Gi"

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  targetCPU: 70

monitoring:
  enabled: true
  sampleRate: 0.1

security:
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000

networkPolicy:
  enabled: true

persistence:
  enabled: true
  size: 10Gi
```

---

## 10. Runbooks

### 10.1 High CPU Usage

```bash
# 1. Check pod status
kubectl top pods -n sre-agent

# 2. Describe problematic pod
kubectl describe pod <pod-name> -n sre-agent

# 3. Check logs
kubectl logs <pod-name> -n sre-agent --previous

# 4. If needed, restart
kubectl rollout restart deployment/sre-agent -n sre-agent
```

### 10.2 Claude API Failures

```bash
# 1. Check API key validity
kubectl get secret sre-agent-secrets -n sre-agent

# 2. Check rate limits
curl http://sre-agent:8080/api/v1/metrics | grep claude

# 3. Enable fallback mode
kubectl set env deployment/sre-agent FALLBACK_MODE=true -n sre-agent
```

### 10.3 Database Connection Issues

```bash
# 1. Check PostgreSQL status
kubectl exec -it <pod> -n sre-agent -- psql -U postgres -c "SELECT 1"

# 2. Check connection pool
curl http://sre-agent:8080/api/v1/health | jq .db_pool

# 3. Restart application
kubectl rollout restart deployment/sre-agent -n sre-agent
```

---

## Summary

This infrastructure document provides a production-grade setup for the SRE AI Agent. Key takeaways:

1. **High Availability**: Multi-AZ deployment with auto-scaling
2. **Security**: Defense-in-depth with WAF, VPC, and Secrets Manager
3. **Observability**: Full Prometheus/Grafana/Loki stack
4. **CI/CD**: Automated GitHub Actions pipeline
5. **Disaster Recovery**: RTO < 15 min, RPO < 5 min
6. **Cost**: ~$646/month for production (AWS)

For implementation, follow the Kubernetes manifests in `configs/k8s/` and use the Helm chart in `charts/sre-agent`.