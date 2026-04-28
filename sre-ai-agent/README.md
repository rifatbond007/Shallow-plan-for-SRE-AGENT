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

## Component Breakdown

### 1. Log Ingestion Service

- **Purpose**: Collect and parse nginx access/error logs + application logs
- **Implementation**: Go service watching log files or receiving via syslog/Fluentd
- **Parser**: Custom nginx log parser (ngx_http_log_module format)
- **Output**: Normalized log entries with timestamps, severity, error codes

### 2. Codebase Analysis Service

- **Purpose**: Read and understand source code to explain behavior
- **Implementation**: Go AST-based analyzer or integrate with tree-sitter
- **Features**:
  - Function call analysis
  - Error handling path tracing
  - Dependency mapping

### 3. AI Analysis Engine

- **Purpose**: Generate hypotheses from log patterns + code context
- **Implementation**:
  - Local: Embeddings + vector similarity (ortex)
  - External: Anthropic Claude API integration
- **Features**:
  - Pattern matching against known error signatures
  - Root cause hypothesis generation
  - Anomaly detection (statistical)

### 4. API Server

- **Purpose**: Expose functionality via REST API
- **Endpoints**:
  - `POST /analyze` - Submit logs for analysis
  - `GET /hypotheses/{id}` - Get generated hypotheses
  - `GET /health` - Health check
  - `WS /stream` - Real-time analysis streaming

## Tech Stack

| Component      | Technology         | Rationale             |
|----------------|--------------------|-----------------------|
| Language       | Go 1.21+           | Performance, K8s native |
| Log Processing | Fluentd/Filebeat    | Industry standard      |
| AI Integration | Anthropic SDK      | Claude for reasoning   |
| Vector DB      | LanceDB/pgvector   | Local embeddings      |
| API Framework  | Gin/Echo           | Go web framework      |
| Container      | Docker             | Build images         |
| Orchestration  | Kubernetes         | Production deployment |
| Monitoring     | Prometheus + Grafana | SRE best practices  |

## File Structure

```
sre-ai-agent/
├── cmd/
│   ├── agent/main.go          # Main entrypoint
│   └── ingestor/main.go       # Log ingestor
├── internal/
│   ├── analyzer/              # AI analysis engine
│   ├── parser/                # Log parsers
│   ├── codebase/              # Code analysis
│   ├── api/                   # HTTP server
│   └── storage/              # Data persistence
├── pkg/                       # Reusable packages
├── configs/
│   ├── docker-compose.yml     # Local dev
│   ├── k8s/                   # Kubernetes manifests
│   └── nginx/                 # Sample nginx config
├── prompts/                    # AI system prompts
├── go.mod
├── go.sum
├── Dockerfile
├── Makefile
└── README.md
```

## Implementation Phases

### Phase 1: Foundation (Week 1-2)

- Project scaffolding with Go modules
- Basic log parser for nginx (access.log)
- Simple REST API with Gin
- Docker Compose for local dev

### Phase 2: Log Processing (Week 3)

- Application log parser
- Log normalization pipeline
- Error classification engine

### Phase 3: Code Analysis (Week 4)

- Codebase reader service
- Basic AST analysis
- Link errors to source locations

### Phase 4: AI Integration (Week 5-6)

- Anthropic Claude SDK integration
- Hypothesis generation prompts
- RAG pipeline with code context

### Phase 5: Kubernetes (Week 7)

- K8s manifests (Deployment, Service, ConfigMap)
- Helm chart
- Health checks, liveness probes

### Phase 6: Polish (Week 8)

- Prometheus metrics
- Grafana dashboards
- Documentation

## Prerequisites

- Go 1.21 or higher
- Docker 24+ and Docker Compose
- Kubernetes 1.28+ (for production)
- Anthropic API key (for AI features)

## Quick Start

### Local Development

```bash
# Clone and setup
git clone https://github.com/yourusername/sre-ai-agent.git
cd sre-ai-agent

# Copy environment template
cp .env.example .env
# Edit .env and add your ANTHROPIC_API_KEY

# Start with Docker Compose
docker-compose up -d

# The API will be available at http://localhost:8080
```

### Kubernetes Deployment

```bash
# Apply Kubernetes manifests
kubectl apply -f configs/k8s/

# Or use Helm (if charts created)
helm install sre-agent ./charts/sre-agent
```

## API Reference

| Endpoint               | Method | Description                |
|------------------------|--------|----------------------------|
| /api/v1/analyze        | POST   | Submit logs for analysis   |
| /api/v1/hypotheses/:id | GET    | Get generated hypotheses   |
| /api/v1/health         | GET    | Health check               |
| /api/v1/stream         | WS     | Real-time streaming        |
| /api/v1/metrics        | GET    | Prometheus metrics         |

## Development

### Setup

```bash
# Install dependencies
go mod download

# Build binary
make build

# Run locally
make run

# Test
make test
```

### Testing

```bash
# Test nginx log parsing
curl -X POST http://localhost:8080/analyze -d @sample.json

# Verify hypotheses response contains AI-generated insights
```

## Deployment

### Kubernetes

```bash
# Apply manifests
kubectl apply -f k8s/

# Check pods running
kubectl get pods -l app=sre-agent
```