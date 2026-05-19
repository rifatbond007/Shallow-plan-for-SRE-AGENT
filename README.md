# SRE AI Agent

An AI-powered Site Reliability Engineering agent that analyzes nginx and application logs, examines codebase, and generates root cause hypotheses.

## Overview

Build an SRE AI Agent as a capstone project that:
1. Parses nginx and application logs
2. Analyzes codebase and generates hypotheses for issues
3. Debug errors automatically
4. Deploy with Docker + Kubernetes

This is a greenfield project - designing the full architecture from scratch.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     SRE AI Agent System                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │   Nginx     │    │  App Logs   │    │   Codebase          │  │
│  │   Logs      │    │  (Go)       │    │   (Analysis)        │  │
│  └──────┬──────┘    └──────┬──────┘    └──────────┬──────────┘  │
│         │                  │                       │             │
│         └────────┬─────────┘                       │             │
│                  ▼                                 ▼             │
│         ┌────────────────┐               ┌──────────────────┐    │
│         │ Log Ingestion  │               │ Codebase Reader  │    │
│         │ Service        │               │ Service          │    │
│         └───────┬────────┘               └────────┬─────────┘    │
│                 │                                 │              │
│                 ▼                                 ▼              │
│         ┌──────────────────────────────────────────────────┐    │
│         │            Analysis Engine (AI/ML)                │    │
│         │  - Pattern Recognition                            │    │
│         │  - Error Classification                          │    │
│         │  - Hypothesis Generation                         │    │
│         │  - Root Cause Analysis                           │    │
│         └───────────────────────┬──────────────────────────┘    │
│                                 │                               │
│                                 ▼                               │
│         ┌──────────────────────────────────────────────────┐    │
│         │              API Server (Go)                      │    │
│         │  - REST API for queries                          │    │
│         │  - WebSocket for streaming                       │    │
│         │  - Alerting endpoints                            │    │
│         └──────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Tech Stack

| Component      | Technology         | Rationale             |
|----------------|--------------------|-----------------------|
| Language       | Go 1.21+           | Performance, K8s native |
| Log Processing | Fluentd/Filebeat   | Industry standard     |
| AI Integration | Anthropic SDK      | Claude for reasoning  |
| Vector DB      | LanceDB/pgvector   | Local embeddings      |
| API Framework  | Gin/Echo           | Go web framework      |
| Container      | Docker             | Build images          |
| Orchestration  | Kubernetes         | Production deployment |
| Monitoring     | Prometheus + Grafana | SRE best practices  |

## Quick Start

```bash
# Clone the repository
git clone https://github.com/rifatbond007/Shallow-plan-for-SRE-AGENT.git
cd Shallow-plan-for-SRE-AGENT/sre-ai-agent

# Local with Docker
docker-compose up -d

# Kubernetes
helm install sre-agent ./charts/sre-agent
```

## API Reference

| Endpoint          | Method   | Description                |
|-------------------|----------|----------------------------|
| /api/v1/analyze   | POST     | Submit logs for analysis   |
| /api/v1/hypotheses/:id | GET | Get generated hypotheses   |
| /api/v1/health    | GET      | Health check               |
| /api/v1/stream    | WS       | Real-time streaming        |
| /api/v1/metrics   | GET      | Prometheus metrics         |

## Documentation

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](sre-ai-agent/ARCHITECTURE.md) | System architecture diagrams and component details |
| [FOLDER_STRUCTURE.md](sre-ai-agent/FOLDER_STRUCTURE.md) | Project folder structure and organization |
| [PLAN.md](sre-ai-agent/plan.md) | 10-week implementation plan |
| [COST.md](sre-ai-agent/COST.md) | Monthly operational cost breakdown |
| [INFRASTRUCTURE.md](sre-ai-agent/INFRASTRUCTURE.md) | Cloud infrastructure and deployment |
| [DEVOPS.md](sre-ai-agent/DEVOPS.md) | DevOps operations handbook with implementation guidelines |

## Development

### Setup

```bash
# Install dependencies
go mod download

# Run locally
make run

# Test
make test
```

### Testing

```bash
# Test nginx log parsing
curl -X POST http://localhost:8080/api/v1/analyze -d @sample.json

# Verify hypotheses response contains AI-generated insights
```

## Deployment

### Kubernetes

```bash
# Apply manifests
kubectl apply -f sre-ai-agent/configs/k8s/

# Check pods running
kubectl get pods -l app=sre-agent
```

## License

MIT