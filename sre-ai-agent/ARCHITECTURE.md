# SRE AI Agent - Architecture

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                     SRE AI AGENT SYSTEM                                       │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                                    DATA SOURCES                                     │    │
│  │  ┌──────────────────┐   ┌──────────────────┐   ┌──────────────────────────────┐  │    │
│  │  │   NGINX          │   │   APPLICATION    │   │     CODEBASE                 │  │    │
│  │  │   - access.log   │   │   - app.log      │   │     - Go source files        │  │    │
│  │  │   - error.log    │   │   - debug.log    │   │     - Error handlers         │  │    │
│  │  │   - metrics      │   │   - audit.log    │   │     - Function definitions   │  │    │
│  │  └────────┬─────────┘   └────────┬─────────┘   └──────────────┬───────────────┘  │    │
│  │           │                       │                            │                   │    │
│  └───────────┼───────────────────────┼────────────────────────────┼───────────────────┘    │
│              │                       │                            │                        │
│              ▼                       ▼                            ▼                        │
│  ┌────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                               LOG INGESTION LAYER                                   │    │
│  │  ┌──────────────────────────────────────────────────────────────────────────────┐  │    │
│  │  │                          Log Ingestion Service                                │  │    │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │  │    │
│  │  │  │   Nginx     │  │    App      │  │   Log       │  │     File           │ │  │    │
│  │  │  │   Parser    │  │   Parser    │  │  Normalizer │  │     Watcher        │ │  │    │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘ │  │    │
│  │  └──────────────────────────────────────────────────────────────────────────────┘  │    │
│  └────────────────────────────────────────────────────────────────────────────────────┘    │
│              │                                                                       │
│              │ Normalized Logs                                                        │
│              ▼                                                                       │
│  ┌────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                              PROCESSING LAYER                                       │    │
│  │                                                                                      │    │
│  │  ┌─────────────────────────────┐    ┌─────────────────────────────────────────┐   │    │
│  │  │     CODE ANALYSIS SERVICE   │    │         AI ANALYSIS ENGINE              │   │    │
│  │  │  ┌────────────────────────┐ │    │  ┌────────────────────────────────────┐ │   │    │
│  │  │  │   File System Scanner  │ │    │  │     Pattern Recognition           │ │   │    │
│  │  │  ├────────────────────────┤ │    │  ├────────────────────────────────────┤ │   │    │
│  │  │  │   Go AST Parser        │ │    │  │     Error Classification          │ │   │    │
│  │  │  ├────────────────────────┤ │    │  ├────────────────────────────────────┤ │   │    │
│  │  │  │   Error Handler        │ │    │  │     Anomaly Detection             │ │   │    │
│  │  │  │   Analyzer             │ │    │  ├────────────────────────────────────┤ │   │    │
│  │  │  ├────────────────────────┤ │    │  │     Hypothesis Generator          │ │   │    │
│  │  │  │   Function Call        │ │    │  ├────────────────────────────────────┤ │   │    │
│  │  │  │   Graph Builder        │ │    │  │     Claude API Integration        │ │   │    │
│  │  │  ├────────────────────────┤ │    │  ├────────────────────────────────────┤ │   │    │
│  │  │  │   Code-to-Log           │ │    │  │     Root Cause Analyzer           │ │   │    │
│  │  │  │   Linker                │ │    │  └────────────────────────────────────┘ │   │    │
│  │  │  └────────────────────────┘ │    └─────────────────────────────────────────┘   │    │
│  │  └─────────────────────────────┘                                                  │    │
│  └────────────────────────────────────────────────────────────────────────────────────┘    │
│              │                               │                                              │
│              │ Code Context                  │ Analysis Results                            │
│              ▼                               ▼                                              │
│  ┌────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                                 API SERVER LAYER                                    │    │
│  │  ┌──────────────────────────────────────────────────────────────────────────────┐  │    │
│  │  │                              REST API (Gin)                                   │  │    │
│  │  │                                                                               │  │    │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │  │    │
│  │  │  │  POST       │  │   GET       │  │   GET       │  │   WebSocket         │ │  │    │
│  │  │  │  /analyze   │  │  /health    │  │  /hypo-     │  │   /stream           │ │  │    │
│  │  │  │             │  │             │  │  theses/:id │  │                     │ │  │    │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘ │  │    │
│  │  │                                                                               │  │    │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                          │  │    │
│  │  │  │   Middle-   │  │   Request   │  │   Response  │                          │  │    │
│  │  │  │   ware      │  │   Validator │  │   Formatter │                          │  │    │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                          │  │    │
│  │  └──────────────────────────────────────────────────────────────────────────────┘  │    │
│  └────────────────────────────────────────────────────────────────────────────────────┘    │
│              │                                                                       │
│              ▼                                                                       │
│  ┌────────────────────────────────────────────────────────────────────────────────────┐    │
│  │                                 STORAGE LAYER                                       │    │
│  │  ┌──────────────────────────────────────────────────────────────────────────────┐  │    │
│  │  │                           In-Memory Store                                     │  │    │
│  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │  │    │
│  │  │  │  Hypotheses    │  │   Analysis      │  │     Configuration           │  │  │    │
│  │  │  │   Storage      │  │   Results       │  │     Cache                   │  │  │    │
│  │  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │  │    │
│  │  └──────────────────────────────────────────────────────────────────────────────┘  │    │
│  └────────────────────────────────────────────────────────────────────────────────────┘    │
│                                                                                              │
└──────────────────────────────────────────────────────────────────────────────────────────────┘


                                    EXTERNAL INTEGRATIONS

    ┌──────────────────┐              ┌──────────────────┐              ┌──────────────────┐
    │                  │              │                  │              │                  │
    │  Claude API      │              │   Prometheus     │              │   Kubernetes     │
    │  (Anthropic)     │              │   (Metrics)      │              │   (Orchestra-    │
    │                  │              │                  │              │    tion)         │
    └──────────────────┘              └──────────────────┘              └──────────────────┘



                                    DATA FLOW

    ┌──────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
    │Logs  │────▶│ Ingest   │────▶│ Process  │────▶│ Analyze  │────▶│ Response │
    └──────┘     └──────────┘     └──────────┘     └──────────┘     └──────────┘
                 1. Parse          2. Normalize     3. AI Engine      4. Return


                                    COMPONENT INTERACTIONS

    ┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
    │ Client  │────▶│  API    │────▶│Analysis │────▶│ Storage │
    │         │◀────│ Server  │◀────│ Engine  │◀────│         │
    └─────────┘     └─────────┘     └─────────┘     └─────────┘
                            │            │
                            ▼            ▼
                       ┌─────────┐  ┌─────────┐
                       │ Code    │  │ Claude  │
                       │ Analyzer│  │ API     │
                       └─────────┘  └─────────┘
```

## Component Details

### 1. Data Sources Layer
| Component | Description | Format |
|-----------|-------------|--------|
| Nginx Access Log | HTTP request logs | Combined log format |
| Nginx Error Log | Server errors | Error messages |
| Application Log | Go application logs | JSON structured |
| Codebase | Source code for analysis | Go files (.go) |

### 2. Log Ingestion Layer
| Component | Responsibility |
|-----------|----------------|
| Nginx Parser | Parse combined log format, extract IP, timestamp, request, status |
| App Parser | Parse JSON structured logs, extract level, message, stack trace |
| Log Normalizer | Convert all logs to unified format with severity detection |
| File Watcher | Monitor log files in real-time (tail -f functionality) |

### 3. Processing Layer

#### Code Analysis Service
| Component | Responsibility |
|-----------|----------------|
| File Scanner | Recursively scan codebase directory |
| AST Parser | Parse Go source into Abstract Syntax Tree |
| Error Analyzer | Identify error handling patterns |
| Call Graph Builder | Build function call relationships |
| Code-to-Log Linker | Connect log errors to source locations |

#### AI Analysis Engine
| Component | Responsibility |
|-----------|----------------|
| Pattern Recognition | Match logs against known error signatures |
| Error Classification | Categorize errors (network, database, auth, etc.) |
| Anomaly Detection | Statistical analysis for unusual patterns |
| Hypothesis Generator | Generate root cause hypotheses |
| Claude Integration | LLM-powered deep analysis |
| Root Cause Analyzer | Determine probable cause chains |

### 4. API Server Layer
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analyze` | POST | Submit logs for analysis |
| `/api/v1/hypotheses/:id` | GET | Get hypothesis by ID |
| `/api/v1/health` | GET | Health check |
| `/api/v1/stream` | WS | Real-time analysis stream |
| `/api/v1/metrics` | GET | Prometheus metrics |

### 5. Storage Layer
| Component | Description |
|-----------|-------------|
| Hypotheses Storage | Store generated hypotheses (in-memory by default) |
| Analysis Results | Cache analysis results |
| Configuration Cache | Cache parsed configurations |
| PostgreSQL | Optional - for persistent storage |
| Redis | Optional - for caching and rate limiting |

## Deployment Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     KUBERNETES CLUSTER                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    sre-agent Deployment                  │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐        │   │
│  │  │   Pod 1    │  │   Pod 2    │  │   Pod 3    │        │   │
│  │  │ sre-agent  │  │ sre-agent  │  │ sre-agent  │        │   │
│  │  │    :8080   │  │    :8080   │  │    :8080   │        │   │
│  │  └────────────┘  └────────────┘  └────────────┘        │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │                                       │
│                           ▼                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              sre-agent Service (ClusterIP)              │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │                                       │
│                           ▼                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Ingress Controller                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              ConfigMap (configuration)                   │   │
│  │              Secret (API keys)                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘


┌─────────────────────────────────────────────────────────────────┐
│                    LOCAL DEVELOPMENT                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐ │
│  │   SRE Agent │    │  PostgreSQL │    │      Redis          │ │
│  │   :8080     │    │   :5432*    │    │     :6379*          │ │
│  └─────────────┘    └─────────────┘    └─────────────────────┘ │
│         │                * Optional                              │
│         ▼                                                         │
│  ┌─────────────┐                                                  │
│  │  Nginx      │                                                  │
│  │  (source)   │                                                  │
│  └─────────────┘                                                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Request/Response Flow

### Analyze Request Flow
```
Client Request
     │
     ▼
┌─────────────┐
│  Validate   │ ──▶ 400 Bad Request
│  Request    │
└──────┬──────┘
       │ Valid
       ▼
┌─────────────┐
│  Parse Logs │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│  Analyze    │────▶│ Claude API  │
│  with Code  │     │ (if needed) │
│  Context    │     └──────┬──────┘
└──────┬──────┘            │
       │                   │
       ▼                   │
┌─────────────┐            │
│  Generate   │◀───────────┘
│  Hypotheses │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Store      │
│  Results    │
└──────┬──────┘
       │
       ▼
    Response
```

## Error Handling Flow
```
Log Entry
    │
    ▼
┌─────────────┐
│  Parse &    │
│  Normalize  │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│  Known      │────▶│ Execute     │
│  Pattern    │     │ Fix         │
│  Match?     │     └─────────────┘
└──────┬──────┘
       │ No
       ▼
┌─────────────┐     ┌─────────────┐
│  Anomaly    │────▶│ Alert       │
│  Detected?  │     │ Admin       │
└──────┬──────┘     └─────────────┘
       │ No
       ▼
┌─────────────┐     ┌─────────────┐
│  Use Claude │────▶│ Generate    │
│  API        │     │ Hypothesis  │
└─────────────┘     └─────────────┘
```

## Technology Stack Summary

| Layer | Technology | Version |
|-------|------------|---------|
| Language | Go | 1.21+ |
| Web Framework | Gin | latest |
| AI SDK | Anthropic | latest |
| Container | Docker | 24+ |
| Orchestration | Kubernetes | 1.28+ |
| Monitoring | Prometheus | 2.45+ |
| Visualization | Grafana | 10+ |