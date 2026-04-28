# SRE AI Agent - Folder Structure

## Root Directory Structure

```
sre-ai-agent/
├── cmd/                        # Application entrypoints
├── internal/                   # Private application code
├── pkg/                        # Reusable public packages
├── configs/                    # Configuration files
├── prompts/                    # AI system prompts
├── scripts/                    # Utility scripts
├── docs/                       # Documentation
├── tests/                      # Test data and helpers
├── charts/                     # Helm charts
├── go.mod                      # Go module definition
├── go.sum                      # Go dependencies checksum
├── Makefile                    # Build automation
├── Dockerfile                  # Container image definition
├── docker-compose.yml          # Local development setup
├── .gitignore                   # Git ignore patterns
├── .env.example                # Environment variables template
├── ARCHITECTURE.md             # Architecture diagram
├── FOLDER_STRUCTURE.md         # This file
├── plan.md                     # 10-week implementation plan
└── README.md                   # Main documentation
```

---

## Detailed Folder Structure

```
sre-ai-agent/
│
├── cmd/
│   │
│   └── agent/
│       ├── main.go              # Main application entrypoint
│       └── wire.go              # Dependency injection (optional)
│
├── internal/
│   │
│   ├── analyzer/
│   │   │
│   │   ├── engine.go            # Main analysis engine
│   │   ├── patterns.go           # Error pattern matching
│   │   ├── hypothesis.go         # Hypothesis generation
│   │   ├── anomaly.go           # Anomaly detection
│   │   └── claude.go            # Claude API client
│   │
│   ├── api/
│   │   │
│   │   ├── server.go            # HTTP server setup
│   │   ├── handlers.go          # Request handlers
│   │   ├── middleware.go        # HTTP middleware
│   │   ├── validators.go        # Request validation
│   │   ├── responses.go         # Response formatters
│   │   └── routes.go            # Route definitions
│   │
│   ├── codebase/
│   │   │
│   │   ├── reader.go            # File system scanner
│   │   ├── parser.go            # Go AST parser
│   │   ├── analyzer.go          # Code analysis logic
│   │   ├── callgraph.go         # Function call graph
│   │   ├── errors.go            # Error pattern finder
│   │   └── linker.go            # Code-to-log linker
│   │
│   ├── parser/
│   │   │
│   │   ├── nginx.go             # Nginx log parser
│   │   ├── nginx_access.go     # Access log parser
│   │   ├── nginx_error.go      # Error log parser
│   │   ├── app.go              # Application log parser
│   │   ├── normalizer.go       # Log normalization
│   │   └── watcher.go          # File watcher
│   │
│   ├── storage/
│   │   │
│   │   ├── store.go            # In-memory store
│   │   ├── hypothesis.go        # Hypothesis storage
│   │   ├── analysis.go          # Analysis results
│   │   └── cache.go             # Configuration cache
│   │
│   └── config/
│       │
│       ├── config.go            # Configuration struct
│       ├── loader.go            # Config loading
│       └── defaults.go          # Default values
│
├── pkg/
│   │
│   ├── logger/
│   │   │
│   │   ├── logger.go           # Logging interface
│   │   └── zap.go              # Zap logger implementation
│   │
│   ├── metrics/
│   │   │
│   │   ├── metrics.go          # Prometheus metrics
│   │   └── counters.go         # Custom counters
│   │
│   └── utils/
│       │
│       ├── errors.go            # Error utilities
│       ├── strings.go           # String utilities
│       └── time.go              # Time utilities
│
├── configs/
│   │
│   ├── docker-compose.yml       # Local development
│   ├── docker-compose.prod.yml  # Production setup
│   │
│   ├── k8s/
│   │   │
│   │   ├── deployment.yaml      # K8s deployment
│   │   ├── service.yaml         # K8s service
│   │   ├── configmap.yaml       # ConfigMap
│   │   ├── secret.yaml          # Secrets
│   │   ├── ingress.yaml         # Ingress
│   │   ├── pvc.yaml             # Persistent volume
│   │   └── hpa.yaml             # Horizontal pod autoscaler
│   │
│   └── nginx/
│       │
│       ├── nginx.conf           # Nginx configuration
│       └── default.conf         # Default server config
│
├── charts/
│   │
│   └── sre-agent/
│       │
│       ├── Chart.yaml           # Helm chart metadata
│       ├── values.yaml          # Default values
│       ├── values.prod.yaml     # Production values
│       ├── values.staging.yaml  # Staging values
│       └── templates/
│           │
│           ├── deployment.yaml
│           ├── service.yaml
│           ├── configmap.yaml
│           ├── secret.yaml
│           └── ingress.yaml
│
├── prompts/
│   │
│   ├── system.txt               # System prompt for AI
│   ├── analysis.txt             # Analysis prompt
│   ├── hypothesis.txt           # Hypothesis generation
│   └── root-cause.txt           # Root cause analysis
│
├── scripts/
│   │
│   ├── build.sh                # Build script
│   ├── test.sh                 # Test runner
│   ├── deploy.sh               # Deployment script
│   └── cleanup.sh              # Cleanup script
│
├── docs/
│   │
│   ├── api.md                  # API documentation
│   ├── deployment.md           # Deployment guide
│   └── troubleshooting.md      # Troubleshooting guide
│
└── tests/
    │
    ├── data/
    │   │
    │   ├── logs/
    │   │   ├── nginx-access.log # Sample nginx access
    │   │   ├── nginx-error.log  # Sample nginx error
    │   │   └── app.log           # Sample app log
    │   │
    │   └── code/
    │       ├── main.go           # Sample Go code
    │       └── handler.go        # Sample handler
    │
    └── integration/
        │
        └── api_test.go          # API integration tests
```

---

## Folder Purpose Guide

### `/cmd`

Contains entrypoints for executables. Each subdirectory is a separate executable.

```
cmd/
└── agent/
    └── main.go     # The main sre-agent application
```

**When to modify**: Adding new commands or changing the main entrypoint.

---

### `/internal`

Contains private application code that should not be imported outside the project.

```
internal/
├── analyzer/       # AI analysis engine - pattern matching, hypothesis generation
├── api/           # HTTP server - handlers, routes, middleware
├── codebase/      # Code analysis - AST parsing, error finding
├── parser/        # Log parsers - nginx, app logs
├── storage/       # Data storage - in-memory store
└── config/        # Configuration - app settings
```

**When to modify**: Adding new features, fixing bugs in core functionality.

---

### `/pkg`

Contains public packages that can be imported by external projects.

```
pkg/
├── logger/        # Logging utilities
├── metrics/       # Prometheus metrics
└── utils/         # General utilities
```

**When to modify**: Creating reusable code that might be shared.

---

### `/configs`

Configuration files for various environments.

```
configs/
├── docker-compose.yml    # Local dev environment
├── k8s/                 # Kubernetes manifests
└── nginx/               # Nginx configurations
```

**When to modify**: Changing deployment configuration.

---

### `/prompts`

Text prompts used for AI analysis.

```
prompts/
├── system.txt       # Main system prompt
├── analysis.txt     # Log analysis prompt
├── hypothesis.txt   # Hypothesis prompt
└── root-cause.txt   # Root cause analysis
```

**When to modify**: Tuning AI behavior, adding new prompt templates.

---

## File Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Go source files | lowercase, descriptive | `analyzer.go`, `parser.go` |
| Test files | `_test.go` suffix | `analyzer_test.go` |
| Config files | lowercase, hyphenated | `docker-compose.yml` |
| Documentation | lowercase, hyphenated | `api-reference.md` |

---

## Import Structure

```
Standard library
    │
    ▼
External packages (go.mod)
    │
    ▼
Internal packages (./internal/...)
    │
    ▼
Package utilities (./pkg/...)
```

---

## Quick Reference

| Need to... | Look in... |
|------------|-------------|
| Change API endpoints | `internal/api/routes.go` |
| Add new log parser | `internal/parser/` |
| Modify AI analysis | `internal/analyzer/` |
| Change code analysis | `internal/codebase/` |
| Update configuration | `internal/config/` |
| Add Kubernetes resources | `configs/k8s/` |
| Modify AI prompts | `prompts/` |

---

## Development Workflow

1. **Core logic**: Modify files in `internal/`
2. **Public utilities**: Add to `pkg/`
3. **Configuration**: Update in `configs/`
4. **Testing**: Add test data to `tests/data/`
5. **Documentation**: Update docs in root or `docs/`

---

## Building

```bash
# Build binary
make build

# Run locally
make run

# Run tests
make test

# Build Docker image
make docker-build
```

---

## Environment Variables

Create `.env` from `.env.example`:

```bash
# Copy template
cp .env.example .env

# Edit with your values
```

Example `.env.example`:
```bash
# API Configuration
API_PORT=8080
LOG_LEVEL=info

# Anthropic AI
ANTHROPIC_API_KEY=sk-ant-your-key-here

# Codebase Analysis
CODEBASE_PATH=/app/codebase

# Log Configuration
LOG_DIR=/var/log

# Optional: Redis for caching
REDIS_URL=redis://localhost:6379

# Optional: PostgreSQL for persistence
DATABASE_URL=postgres://user:pass@localhost:5432/sre_agent
```

## Kubernetes Deployment

```bash
# Apply configurations
kubectl apply -f configs/k8s/

# Or use Helm
helm install sre-agent ./charts/sre-agent

# Check status
kubectl get pods -l app=sre-agent
```