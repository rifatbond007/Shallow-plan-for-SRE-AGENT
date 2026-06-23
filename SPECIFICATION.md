# SRE AI Agent — Technical Specification

> **Status:** Draft v2 (Re-planned)
> **Author:** MD. Rifat Hossain (BAIUST)
> **Date:** 2026-06-23
> **Purpose:** Single source of truth for the entire codebase. Every file, interface, and behavior is defined here before implementation.

---

## 0. Why this version exists

The previous plan (`PROPOSAL_ENHANCED.md`, `ARCHITECTURE.md`, `PLAN.md`, `INFRASTRUCTURE.md`, `DEVOPS.md`) had **scope drift**: it tried to ship Terraform, multi-cloud, ELK, vector DBs, Helm chart repositories, chaos engineering, and production-grade AWS infrastructure — all inside a 10-week university thesis. This is unrealistic and obscures the actual research contribution.

This specification:

1. Narrows the project to a **focused, defensible thesis** — an agent that takes nginx/application logs + a Go codebase and returns **detected errors → ranked root-cause hypotheses → suggested code fixes**.
2. Treats **deployment as a deliverable, not the main point**. Deployment is local (Docker + Minikube/Kind). No AWS/GCP, no Terraform.
3. Treats **evaluation as a first-class deliverable**. We measure hypothesis accuracy against a labeled dataset.
4. Replaces the bag of 6 documents with **one specification** that drives every other doc.

If anything in the other `.md` files contradicts this document, **this document wins**.

---

## 1. Scope and non-goals

### 1.1 In scope

| # | Capability |
|---|------------|
| 1 | Parse nginx access logs (combined format) |
| 2 | Parse nginx error logs |
| 3 | Parse Go application logs (JSON, with optional stack traces) |
| 4 | Detect error/warning log lines and group them into *incidents* |
| 5 | Parse a Go codebase using `go/parser` + `go/ast` and build a function call graph |
| 6 | Link log errors to candidate functions in the codebase (heuristic + optional stack-trace match) |
| 7 | Send a structured prompt (logs + relevant code slices) to the Claude API and receive **ranked root-cause hypotheses** |
| 8 | For the top hypothesis, ask the LLM to suggest a **code-level fix** (function/section-level diff or replacement snippet) |
| 9 | Expose a REST API (Gin) for synchronous analysis and a WebSocket for streaming progress |
| 10 | Expose Prometheus metrics for observability |
| 11 | Run as a single Docker image; deployable to local Kubernetes (Minikube/Kind) via Helm |
| 12 | Provide a reproducible evaluation harness (labeled incidents → accuracy/F1) |

### 1.2 Non-goals (explicitly out of scope)

- Cloud-managed Kubernetes (EKS/GKE/AKS)
- Terraform / IaC
- Multi-cluster / multi-region
- Service mesh (Istio/Linkerd)
- Vector databases (Pinecone, Weaviate, Qdrant, pgvector, LanceDB)
- Production observability stacks (Loki, Tempo, ELK, Datadog)
- PagerDuty / on-call integration
- Auto-remediation (actually patching & deploying code)
- Authentication/authorization beyond a single static API key
- Languages other than Go for the target codebase
- Long-term log retention / archival

If your supervisor pushes for any of the above, add it as a *future work* item, not a v1 requirement.

---

## 2. High-level architecture

### 2.1 One-line description

> An agent that, given a slice of operational logs and a Go codebase, identifies incidents, traces them to candidate functions, and produces ranked root-cause hypotheses with suggested code fixes — all driven by a Claude LLM with deterministic pre/post-processing.

### 2.2 Component map

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         SRE AI Agent (v2)                                │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌──────────────┐    ┌─────────────────┐    ┌──────────────────────┐   │
│   │  Data        │    │  Ingestion      │    │  Codebase            │   │
│   │  Sources     │───▶│  Layer          │    │  Analyzer            │   │
│   │              │    │                 │    │                      │   │
│   │ • nginx      │    │ • nginx access  │    │ • file scanner       │   │
│   │ • nginx err  │    │ • nginx error   │    │ • go/ast parser      │   │
│   │ • app (JSON) │    │ • app JSON      │    │ • call graph         │   │
│   │ • go source  │    │ • normalizer    │    │ • error-linker       │   │
│   └──────────────┘    │ • incident grp. │    └──────────┬───────────┘   │
│                       └────────┬────────┘               │               │
│                                │                        │               │
│                                ▼                        ▼               │
│                       ┌─────────────────────────────────────────┐       │
│                       │         Analysis Engine                  │       │
│                       │  • pattern matcher (deterministic)       │       │
│                       │  • context builder (logs + code slices)  │       │
│                       │  • Claude client (hypothesis + fix)      │       │
│                       │  • ranker / scorer                       │       │
│                       └─────────────────┬───────────────────────┘       │
│                                         │                               │
│                                         ▼                               │
│                       ┌─────────────────────────────────────────┐       │
│                       │         API Server (Gin)                 │       │
│                       │  • REST: /analyze, /hypotheses/:id       │       │
│                       │  • WebSocket: /stream                    │       │
│                       │  • Metrics: /metrics                     │       │
│                       │  • Health: /healthz, /readyz             │       │
│                       └─────────────────┬───────────────────────┘       │
│                                         │                               │
│                                         ▼                               │
│                       ┌─────────────────────────────────────────┐       │
│                       │         Storage (in-process)             │       │
│                       │  • hypothesis cache (bounded LRU)        │       │
│                       │  • analysis results (TTL 1h)             │       │
│                       └─────────────────────────────────────────┘       │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 2.3 Request lifecycle (the heart of the system)

```
HTTP POST /api/v1/analyze  { logs: "...", codebase_path: "/repo" }
   │
   ▼
[1] Validate request
   │
   ▼
[2] Parse logs
   │   • nginx access → AccessLogEntry[]
   │   • nginx error  → ErrorLogEntry[]
   │   • app JSON     → AppLogEntry[]
   ▼
[3] Normalize → unified LogEntry[]
   │
   ▼
[4] Group into Incidents (sliding window + similarity)
   │   • incident = cluster of related errors within time window
   ▼
[5] Codebase analysis (one-shot, cached on disk)
   │   • AST, call graph, function index
   ▼
[6] For each incident:
      a. code-linker: find top-K candidate functions
      b. context-builder: slice relevant code (≤ 8KB)
      c. (deterministic) pattern matcher: check known signatures
      d. (LLM) Claude: hypothesis + fix
      e. scorer: combine pattern + LLM confidence
   │
   ▼
[7] Aggregate AnalysisResult { incidents, hypotheses[], fixes[] }
   │
   ▼
[8] Cache result (LRU, key = hash of input)
   │
   ▼
HTTP 200 { id, summary, incidents: [...], hypotheses: [...] }
```

For long-running analyses, the same flow runs over a WebSocket and emits progress events.

---

## 3. Repository layout

```
sre-ai-agent/
├── cmd/
│   └── agent/
│       └── main.go                 # entrypoint, wires everything
├── internal/
│   ├── config/
│   │   └── config.go               # env + flag loading
│   ├── ingest/
│   │   ├── ingest.go               # public interface
│   │   ├── nginx_access.go         # combined-log-format parser
│   │   ├── nginx_error.go          # nginx error log parser
│   │   ├── app_json.go             # JSON application log parser
│   │   ├── normalizer.go           # LogEntry unified type
│   │   └── incident.go             # time-window clustering
│   ├── codebase/
│   │   ├── scanner.go              # walks Go source dirs
│   │   ├── ast.go                  # go/parser + go/ast wrapper
│   │   ├── callgraph.go            # function-level call graph
│   │   └── linker.go               # log error → function candidates
│   ├── analysis/
│   │   ├── engine.go               # orchestrator
│   │   ├── patterns.go             # deterministic signature matcher
│   │   ├── context.go              # builds LLM prompt context
│   │   ├── claude.go               # Anthropic SDK client
│   │   ├── ranker.go               # combines scores
│   │   └── types.go                # Hypothesis, Fix, AnalysisResult
│   ├── api/
│   │   ├── server.go               # gin engine + middleware
│   │   ├── handlers_analyze.go     # POST /analyze
│   │   ├── handlers_get.go         # GET /hypotheses/:id
│   │   ├── handlers_ws.go          # WS /stream
│   │   ├── handlers_meta.go        # /healthz, /readyz, /metrics
│   │   └── errors.go               # error → HTTP code mapping
│   └── storage/
│       ├── cache.go                # generic TTL+LRU cache
│       └── store.go                # analysis result store
├── pkg/
│   ├── logger/
│   │   └── logger.go               # zap-based structured logger
│   └── metrics/
│       └── metrics.go              # prometheus collectors
├── prompts/
│   ├── system.txt                  # base system prompt
│   ├── hypothesis.txt              # hypothesis generation
│   └── fix.txt                     # fix suggestion
├── configs/
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml
│   ├── k8s/                       # plain manifests (Minikube/Kind)
│   │   ├── namespace.yaml
│   │   ├── configmap.yaml
│   │   ├── secret.yaml             # ANTHROPIC_API_KEY
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml            # optional
│   └── helm/
│       └── sre-agent/
│           ├── Chart.yaml
│           ├── values.yaml
│           ├── values-dev.yaml
│           └── templates/
│               ├── deployment.yaml
│               ├── service.yaml
│               ├── configmap.yaml
│               ├── secret.yaml
│               └── ingress.yaml
├── tests/
│   ├── data/
│   │   ├── logs/
│   │   │   ├── nginx-access.log
│   │   │   ├── nginx-error.log
│   │   │   └── app.log
│   │   └── code/
│   │       └── sample-app/         # small Go app with seeded bugs
│   ├── unit/                       # *_test.go colocated is also OK
│   ├── integration/
│   │   └── api_test.go
│   └── eval/
│       ├── cases.json              # labeled incidents
│       ├── runner.go               # runs all cases
│       └── report.go               # accuracy, top-K, F1
├── scripts/
│   ├── gen_sample_logs.sh
│   └── seed_bugs.sh                # writes known-bad Go files
├── docs/
│   ├── API.md
│   ├── EVAL.md
│   └── DEPLOY.md
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── .env.example
├── ARCHITECTURE.md                 # diagram-only, mirrors §2
├── FOLDER_STRUCTURE.md             # mirrors §3
├── PLAN.md                         # mirrors §7
├── INFRASTRUCTURE.md               # mirrors §8 (local only)
├── DEVOPS.md                       # mirrors §9 (trimmed)
├── COST.md                         # mirrors §10
├── SPECIFICATION.md                # this file
└── README.md
```

---

## 4. Component specifications

### 4.1 `internal/config`

**Responsibility:** load configuration from env vars and a single config file (optional).

**Env vars (all prefixed `SRE_AGENT_`):**

| Var | Type | Default | Notes |
|-----|------|---------|-------|
| `PORT` | int | `8080` | HTTP listen port |
| `LOG_LEVEL` | string | `info` | `debug`/`info`/`warn`/`error` |
| `ANTHROPIC_API_KEY` | string | (required) | Claude API key |
| `ANTHROPIC_MODEL` | string | `claude-sonnet-4-20250514` | see prompt templating |
| `ANTHROPIC_MAX_TOKENS` | int | `2048` | per-request cap |
| `ANTHROPIC_TIMEOUT` | duration | `30s` | |
| `CODEBASE_PATH` | string | `/codebase` | mounted in container |
| `CODEBASE_CACHE_DIR` | string | `/tmp/sre-agent/cache` | AST cache |
| `MAX_LOG_BYTES` | int | `5_000_000` | per request |
| `INCIDENT_WINDOW` | duration | `5m` | grouping window |
| `INCIDENT_MIN_SIZE` | int | `3` | minimum error count |
| `HYPOTHESIS_TOP_K` | int | `3` | returned per incident |
| `CACHE_TTL` | duration | `1h` | |
| `CACHE_MAX_ENTRIES` | int | `512` | |
| `RATE_LIMIT_RPS` | float | `5` | per IP |
| `RATE_LIMIT_BURST` | int | `10` | |

**Public type:**

```go
package config

type Config struct {
    Port               int
    LogLevel           string
    Anthropic          AnthropicConfig
    CodebasePath       string
    CodebaseCacheDir   string
    MaxLogBytes        int
    IncidentWindow     time.Duration
    IncidentMinSize    int
    HypothesisTopK     int
    CacheTTL           time.Duration
    CacheMaxEntries    int
    RateLimitRPS       float64
    RateLimitBurst     int
}

type AnthropicConfig struct {
    APIKey    string
    Model     string
    MaxTokens int
    Timeout   time.Duration
}

func Load() (*Config, error)
```

---

### 4.2 `internal/ingest`

**Responsibility:** turn raw text/bytes into normalized `LogEntry` and group them into `Incident`.

**Public types:**

```go
package ingest

type Source string
const (
    SourceNginxAccess Source = "nginx_access"
    SourceNginxError  Source = "nginx_error"
    SourceAppJSON     Source = "app_json"
)

type Severity string
const (
    SevDebug  Severity = "DEBUG"
    SevInfo   Severity = "INFO"
    SevWarn   Severity = "WARN"
    SevError  Severity = "ERROR"
    SevFatal  Severity = "FATAL"
)

type LogEntry struct {
    ID         string            // sha1(source|raw)
    Timestamp  time.Time         // UTC
    Source     Source
    Severity   Severity
    Message    string
    Fields     map[string]string // normalized fields
    Raw        string            // original line
    StackTrace string            // optional, app_json only
}

type Incident struct {
    ID         string
    StartedAt  time.Time
    EndedAt    time.Time
    Entries    []LogEntry
    Signatures []string // matched signature IDs
    TopError   string   // most-frequent normalized message
    ErrorCount int
    WarnCount  int
}
```

**Interfaces:**

```go
type Parser interface {
    Source() Source
    Parse(reader io.Reader) ([]LogEntry, error)
}

type Normalizer interface {
    Normalize(raw string, src Source) (LogEntry, error)
}

type Grouper interface {
    Group(entries []LogEntry) []Incident
}
```

**Files:**

- `nginx_access.go` — combined-log-format parser. Regex:

  ```
  ^(?P<ip>\S+)\s+\S+\s+\S+\s+\[(?P<ts>[^\]]+)\]\s+"(?P<method>\S+)\s+(?P<uri>\S+)\s+\S+"\s+(?P<status>\d{3})\s+(?P<bytes>\d+)\s+"(?P<ref>[^"]*)"\s+"(?P<ua>[^"]*)"
  ```

  Severity mapping by status: 5xx→ERROR, 4xx→WARN, 3xx→INFO, else INFO.

- `nginx_error.go` — nginx error format:

  ```
  YYYY/MM/DD HH:MM:SS [level] PID#TID: *CID message
  ```

- `app_json.go` — JSON. Recognized keys: `ts`/`time`/`timestamp`, `level`/`severity`, `msg`/`message`, `error`, `stack`/`stacktrace`. Unknown keys go into `Fields`.

- `normalizer.go` — implements `Normalizer`. Wraps the parsers and assigns IDs.

- `incident.go` — `Grouper`:
  1. Sort by timestamp.
  2. Sliding window of `IncidentWindow`.
  3. Within a window, group by `TopError` = most-frequent normalized message.
  4. Drop groups smaller than `IncidentMinSize` (or keep but mark as `low_confidence`).
  5. Extract signatures via pattern matcher (see §4.4).

**Performance target:** 10k log lines parsed in < 1s on a 2-vCPU container.

**Test fixtures:** see `tests/data/logs/`. Each fixture has a `*.expected.json` next to it.

---

### 4.3 `internal/codebase`

**Responsibility:** turn a Go source tree into a queryable index of functions, with call relationships and a way to score "how related is this function to a given log error?".

**Public types:**

```go
package codebase

type Function struct {
    ID         string // pkgpath.FuncName
    PkgPath    string
    Name       string
    Receiver   string
    File       string
    Line       int
    EndLine    int
    Signature  string
    Body       string // full body (for LLM context)
    Calls      []string // callee IDs
    IsExported bool
    Doc        string
}

type Index struct {
    Functions map[string]Function
    ByFile    map[string][]string
    Roots     []string // package-level function IDs
    Built     time.Time
}

type Linker interface {
    CandidateFunctions(inc ingest.Incident, k int) []ScoredFunction
}

type ScoredFunction struct {
    Function Function
    Score    float64
    Reasons  []string
}
```

**Files:**

- `scanner.go` — `filepath.WalkDir`, ignore `vendor/`, `node_modules/`, hidden dirs, `*_test.go` (configurable).

- `ast.go` — for each `.go` file (skipping generated files by `// Code generated ... DO NOT EDIT.`):
  - `parser.ParseFile`
  - Walk AST, collect `*ast.FuncDecl` and methods on types.
  - Extract signature, body (`node.Pos()`–`node.End()`), receiver, doc comment.
  - Build `Function`.

- `callgraph.go` — naive intra-package call graph:
  - For each function, walk its body and find `*ast.CallExpr` where `fun.Fun` resolves to a known function in the index.
  - For unresolved calls (external packages), record the symbol name only.
  - Build adjacency; expose `BFS(from, depth int) []Function`.

- `linker.go` — scoring:
  - **+5** if function name appears in any log message (case-insensitive, snake/camel normalized).
  - **+4** if function doc-comment keyword appears in any log message.
  - **+6** if a stack-trace frame matches the function's `File:Line`.
  - **+3** if function returns/handles an error type mentioned in the log.
  - **+2** if the function is reachable within depth 2 from a function matched above.
  - Return top-K by score, with `Reasons` populated for explainability.

**Cache:** `Index` is JSON-serialized to `CodebaseCacheDir/index.json` keyed by `mtime+hash` of the source tree. On container start, load cache if valid; else rebuild.

**Test fixtures:** `tests/data/code/sample-app/` — a small Go HTTP service with 3 seeded bugs:
- `Bug A`: nil-pointer dereference in handler (panic, no recover)
- `Bug B`: missing timeout on outbound HTTP call
- `Bug C`: SQL injection in repository (returns 500 from DB)

---

### 4.4 `internal/analysis`

**Responsibility:** turn `(Incident, candidate functions)` into `(Hypothesis, Fix)`.

**Public types:**

```go
package analysis

type Hypothesis struct {
    ID            string
    IncidentID    string
    Rank          int
    Title         string            // ≤ 80 chars, one-line
    Summary       string            // 2–4 sentences
    Confidence    float64           // 0.0 – 1.0
    Evidence      []Evidence        // log lines + code refs
    SuspectCode   CodeRef           // primary suspect
    RelatedFuncs  []CodeRef
    PatternHit    *PatternMatch     // nil if none
    LLMReasoning  string            // raw LLM justification
}

type Evidence struct {
    Kind        string // "log" | "code" | "pattern"
    LogEntryID  string
    CodeRef     *CodeRef
    Description string
}

type CodeRef struct {
    File   string
    Line   int
    Snippet string // ≤ 30 lines
}

type Fix struct {
    HypothesisID  string
    Summary       string
    UnifiedDiff   string  // proposed diff (may be partial)
    Replacement   string  // proposed replacement snippet
    Confidence    float64
    Caveats       []string
}

type AnalysisResult struct {
    ID           string
    CreatedAt    time.Time
    Incidents    []ingest.Incident
    Hypotheses   []Hypothesis
    Fixes        []Fix
    Summary      string
    DurationMS   int64
}
```

**Files:**

- `engine.go` — orchestrator. Public:

  ```go
  type Engine interface {
      Analyze(ctx context.Context, req AnalyzeRequest) (*AnalysisResult, error)
      AnalyzeStream(ctx context.Context, req AnalyzeRequest, sink func(ProgressEvent) error) (*AnalysisResult, error)
  }
  type AnalyzeRequest struct {
      Logs         string
      CodebasePath string
      TopK         int
  }
  ```

- `patterns.go` — deterministic signature matcher. Ship a small built-in library:

  ```go
  var DefaultPatterns = []Pattern{
      {ID: "nginx_502",  Severity: "ERROR", Regex: `connect\(\) failed.*upstream`, Label: "Upstream connection failure"},
      {ID: "nginx_504",  Severity: "ERROR", Regex: `upstream timed out`,          Label: "Upstream timeout"},
      {ID: "app_panic",  Severity: "FATAL", Regex: `panic: runtime error:`,       Label: "Runtime panic"},
      {ID: "app_nil",    Severity: "FATAL", Regex: `nil pointer dereference`,     Label: "Nil pointer dereference"},
      {ID: "app_deadline", Severity: "ERROR", Regex: `context deadline exceeded`, Label: "Deadline exceeded"},
      {ID: "db_conn",    Severity: "ERROR", Regex: `connection refused.*postgres`, Label: "Database connection refused"},
      {ID: "db_5xx",     Severity: "ERROR", Regex: `SQLSTATE 5`,                  Label: "PostgreSQL server error"},
  }
  ```

  Returns a `PatternMatch` per incident; the LLM is still called but the prompt is shortened.

- `context.go` — assembles the LLM prompt:
  1. System prompt (from `prompts/system.txt`).
  2. Per-incident block:
     - Top 20 log entries in the incident.
     - Pattern matches (if any).
     - Top-K candidate functions with their bodies (truncated to fit token budget).
     - Call-graph snippet showing how candidates relate.
  3. Asks for ranked hypotheses (JSON) — see §4.4 prompt template.

  Token budgeting: hard cap on combined context of ~30k tokens. If exceeded, drop lowest-scored functions first, then trim log entries to most recent.

- `claude.go` — Anthropic SDK wrapper:
  - Single client, reused.
  - Exponential backoff on 429/5xx (max 3 retries).
  - Streaming via SSE for the `AnalyzeStream` path.
  - Returns parsed JSON or wrapped error.

- `ranker.go` — final scoring:
  - If a pattern matched with high confidence (> 0.9), boost that hypothesis to rank 1.
  - Else, use LLM's reported confidence, adjusted by code-linker score.
  - Output is sorted by `Rank`.

---

### 4.5 `internal/api`

**Responsibility:** HTTP surface.

**Routes:**

| Method | Path | Purpose | Response |
|--------|------|---------|----------|
| `POST` | `/api/v1/analyze` | Synchronous analysis | `AnalysisResult` |
| `GET`  | `/api/v1/hypotheses/:id` | Fetch cached result | `AnalysisResult` |
| `GET`  | `/api/v1/incidents/:id` | Fetch one incident's detail | `IncidentDetail` |
| `GET`  | `/api/v1/healthz` | Liveness | `{status:"ok"}` |
| `GET`  | `/api/v1/readyz`  | Readiness (cache warm, Claude client init) | `{status:"ok"\|"degraded"}` |
| `GET`  | `/metrics` | Prometheus | text |
| `WS`   | `/api/v1/stream`  | Streaming analysis | events |

**Files:**

- `server.go` — gin engine construction, middleware chain (recovery → logging → CORS → request-id → ratelimit).

- `handlers_analyze.go` — body limits via `c.Request.Body = http.MaxBytesReader(w, body, MaxLogBytes)`. Returns `400` on bad JSON, `413` on too-large, `502` on Claude error, `504` on timeout, `500` on internal.

- `handlers_get.go` — looks up result by ID in `storage`. Returns `404` if expired.

- `handlers_ws.go` — gorilla/websocket upgrade. Sends events:
  ```json
  { "type": "progress", "stage": "parsing", "pct": 10 }
  { "type": "incident", "data": {...} }
  { "type": "hypothesis", "data": {...} }
  { "type": "done", "result_id": "..." }
  { "type": "error", "message": "..." }
  ```

- `handlers_meta.go` — `/healthz` always 200; `/readyz` pings Claude (cheap model, max_tokens=8) every 60s and caches status.

- `errors.go` — central error → HTTP mapping. Implements `httperror` interface.

**Rate limiting:** in-process token bucket per IP (no Redis dependency). Default 5 RPS, burst 10. Per `RATE_LIMIT_*` env vars.

---

### 4.6 `internal/storage`

**Responsibility:** keep recent results hot.

- `cache.go` — generic LRU + TTL. Uses `container/list` and a map. No external deps.
- `store.go` — wraps cache with `analysis.AnalysisResult`. Key = `result_id` (uuid v4 from request).

No database. No file persistence. Result of a thesis demo is "I ran it, here is the JSON." A future version can add Postgres.

---

### 4.7 `pkg/logger` & `pkg/metrics`

- `pkg/logger/logger.go` — wraps `zap` (sugared). Methods: `Debug`, `Info`, `Warn`, `Error`, `With(fields ...)`. JSON output in prod, console in dev.

- `pkg/metrics/metrics.go` — Prometheus collectors:

  | Name | Type | Labels |
  |------|------|--------|
  | `sre_agent_http_requests_total` | counter | `method`, `path`, `status` |
  | `sre_agent_http_request_duration_seconds` | histogram | `method`, `path` |
  | `sre_agent_analyze_total` | counter | `status` |
  | `sre_agent_analyze_duration_seconds` | histogram | — |
  | `sre_agent_incidents_total` | counter | `source` |
  | `sre_agent_hypotheses_generated_total` | counter | `pattern_hit` |
  | `sre_agent_claude_api_calls_total` | counter | `model`, `status` |
  | `sre_agent_claude_api_errors_total` | counter | `kind` |
  | `sre_agent_claude_tokens_total` | counter | `model`, `direction` |
  | `sre_agent_active_websockets` | gauge | — |

---

## 5. Prompt templates

> LLM behaviour is dictated by the contents of `prompts/*.txt`. These are part of the deliverable, not free-text.

### 5.1 `prompts/system.txt`

```
You are an SRE assistant. Given (a) a cluster of error log lines from a running
service and (b) the relevant Go source code, you produce:

1. A ranked list of root-cause hypotheses, each with:
   - title: <one line, <= 80 chars>
   - summary: <2-4 sentences>
   - confidence: <0.0 to 1.0>
   - suspect_function: <fully qualified function name>
   - evidence_log_ids: <ids of the log lines that support this>
2. For the top hypothesis, a proposed code fix:
   - summary: <one line>
   - replacement: <a Go function body or patch that resolves the issue>
   - caveats: <list of assumptions>

You must respond in strict JSON matching the schema given. Do not include
prose outside the JSON. If the evidence is insufficient, return
confidence <= 0.3 and say so in the summary.
```

### 5.2 `prompts/hypothesis.txt`

```
INCIDENT #{incident.id}
Window: {incident.started_at} -> {incident.ended_at}
Errors: {incident.error_count}, Warnings: {incident.warn_count}
Top message: {incident.top_error}
Pattern matches: {incident.signatures}

LOG ENTRIES (most recent first):
{formatted_log_entries}

CANDIDATE FUNCTIONS (ranked by code-linker score):
{for each function:
  - {function.id}  ({function.file}:{function.line})
  Signature: {function.signature}
  Body:
  ```go
  {function.body}
  ```
  Called by: {function.callers_summary}
  Calls:     {function.callees_summary}
}

Return a JSON object of the form:
{
  "hypotheses": [
    {
      "rank": 1,
      "title": "...",
      "summary": "...",
      "confidence": 0.0,
      "suspect_function": "pkg.Func",
      "evidence_log_ids": ["..."]
    }
  ]
}
```

### 5.3 `prompts/fix.txt`

```
Top hypothesis:
{top_hypothesis_json}

Function to patch: {function.id} at {file}:{line}
Current body:
```go
{function.body}
```

Produce a JSON object:
{
  "summary": "...",
  "replacement": "<full new function body, valid Go>",
  "unified_diff": "<optional unified diff>",
  "caveats": ["..."]
}
```

---

## 6. Data model — wire formats

### 6.1 Request: `POST /api/v1/analyze`

```json
{
  "logs": "<raw multi-line string; auto-detected per line>",
  "codebase_path": "/codebase",
  "top_k": 3
}
```

### 6.2 Response: `200 OK`

```json
{
  "id": "9f0c...",
  "created_at": "2026-06-23T12:34:56Z",
  "duration_ms": 4210,
  "summary": "1 incident: upstream timeout in checkout handler",
  "incidents": [
    {
      "id": "inc_01H...",
      "started_at": "...",
      "ended_at": "...",
      "error_count": 17,
      "warn_count": 3,
      "top_error": "context deadline exceeded",
      "signatures": ["app_deadline"]
    }
  ],
  "hypotheses": [
    {
      "id": "hyp_01H...",
      "incident_id": "inc_01H...",
      "rank": 1,
      "title": "Missing HTTP client timeout in checkout handler",
      "summary": "...",
      "confidence": 0.86,
      "evidence": [
        { "kind": "log", "log_entry_id": "abc", "description": "..." }
      ],
      "suspect_code": {
        "file": "/codebase/internal/checkout/handler.go",
        "line": 42,
        "snippet": "..."
      },
      "pattern_hit": { "id": "app_deadline", "label": "Deadline exceeded" }
    }
  ],
  "fixes": [
    {
      "hypothesis_id": "hyp_01H...",
      "summary": "Add 3s timeout to outbound call",
      "replacement": "...",
      "unified_diff": "--- a/...\n+++ b/...\n@@ ...",
      "confidence": 0.78,
      "caveats": ["Requires testing in staging"]
    }
  ]
}
```

### 6.3 Error responses

```json
{ "error": { "code": "INVALID_LOGS", "message": "logs field is empty" } }
```

Codes: `INVALID_REQUEST`, `LOGS_TOO_LARGE`, `CODEBASE_NOT_FOUND`, `CLAUDE_UPSTREAM_ERROR`, `TIMEOUT`, `INTERNAL`.

---

## 7. Implementation roadmap (phased)

This is the realistic plan. The previous 10-week flat plan is replaced by 3 phases; the thesis demo is the end of Phase 2.

### Phase 0 — Foundation (Week 1)
- Repo scaffold (`go mod init`, Makefile, Dockerfile, .env.example).
- `pkg/logger`, `pkg/metrics`, `internal/config`.
- `cmd/agent/main.go` boots a "hello" HTTP server with `/healthz` and `/readyz`.
- Docker image builds and runs.

### Phase 1 — MVP: detect + hypothesize (Weeks 2–4)
- All three log parsers + normalizer + grouper, with fixtures and unit tests.
- `internal/codebase` (scanner, AST, call graph, linker) with sample-app.
- Deterministic pattern matcher.
- Claude client + hypothesis prompt + structured output.
- `POST /api/v1/analyze` end-to-end.
- `GET /hypotheses/:id`.
- Eval harness with 10 labeled cases (Bug A, B, C + 7 generated).

**Demo gate:** "Give it a log, get a ranked hypothesis with a code ref."

### Phase 2 — V1: add fix suggestion (Weeks 5–6)
- Fix prompt + parser.
- Top-hypothesis → `Fix` block in response.
- Code slicing improvements (call-graph neighbor context).
- WebSocket streaming.
- Prometheus metrics + Grafana dashboard JSON.

**Demo gate:** "Give it a log, get hypothesis AND a proposed code patch."

### Phase 3 — V2: deployment & polish (Weeks 7–8)
- docker-compose for local dev with sample-app.
- K8s manifests (plain YAML) for Minikube/Kind.
- Helm chart.
- README, docs/API.md, docs/EVAL.md, docs/DEPLOY.md.
- 20+ labeled eval cases, accuracy ≥ 0.7 on top-1.

**Demo gate:** Reproducible, documented, deployable. Thesis submission.

### Stretch (only if everything else is done)
- Multi-repo codebase support.
- Incremental log streaming.
- A web UI (read-only) showing the analysis result.

---

## 8. Local deployment

> Cloud deployment is out of scope. Everything below is reproducible on a laptop.

### 8.1 Container

- **Base:** `golang:1.22-alpine` (build) → `alpine:3.20` (run).
- **Multi-stage build**, CGO disabled, stripped binary.
- Non-root user (`appuser`, uid 10001).
- Read-only root FS, writable `/tmp` and `/cache` volumes.
- Healthcheck via `/api/v1/healthz`.
- Image size target: < 50 MB.

### 8.2 docker-compose (dev)

```
services:
  sre-agent:
    build: .
    ports: ["8080:8080"]
    env_file: .env
    volumes:
      - ./tests/data/code/sample-app:/codebase:ro
      - sre-cache:/tmp/sre-agent/cache
  prometheus:
    image: prom/prometheus:v2.55.0
    ports: ["9090:9090"]
    volumes: ["./configs/prometheus.yml:/etc/prometheus/prometheus.yml:ro"]
  grafana:
    image: grafana/grafana:11.2.0
    ports: ["3000:3000"]
volumes:
  sre-cache:
```

### 8.3 Kubernetes (Minikube / Kind)

- Single Deployment, 2 replicas, resource requests/limits.
- Secret for `ANTHROPIC_API_KEY`.
- ConfigMap for non-secret env.
- Service `ClusterIP`.
- Optional Ingress (nginx-ingress).
- **No** HPA, PDB, NetworkPolicy in v1 (we're on a laptop).
- Helm chart in `configs/helm/sre-agent/`.

### 8.4 Bootstrap script (for the demo)

`scripts/demo.sh`:
1. Start `docker-compose up -d`.
2. Wait for `/healthz`.
3. `curl` the seeded logs into `/analyze`.
4. `curl` the eval runner (`go test ./tests/eval/...`).
5. Print pass/fail.

---

## 9. Evaluation methodology (a thesis is judged on this)

### 9.1 Dataset

- **10–20 labeled incidents**, each consisting of:
  - A snippet of logs (real or synthesized).
  - The Go source tree they came from.
  - Ground-truth: `incident_type`, `root_cause_function`, `fix_summary`.
- Mix of:
  - The 3 seeded bugs in `sample-app/`.
  - Synthetic incidents generated by mutating a known-good Go service (e.g., remove a nil check, add an infinite loop, swap a config value).
  - 2–3 hand-crafted incidents inspired by real-world nginx + Go bugs (5xx cascade, slow upstream, DB connection pool exhaustion).

Stored in `tests/eval/cases.json`:
```json
[
  {
    "id": "case_001",
    "name": "nil pointer in handler",
    "logs_file": "tests/data/eval/case_001.log",
    "codebase_path": "tests/data/code/sample-app",
    "ground_truth": {
      "function": "github.com/sample/handler.GetUser",
      "fix_summary": "Add nil check on user before dereference"
    }
  }
]
```

### 9.2 Metrics

| Metric | Definition | Target |
|--------|------------|--------|
| **Top-1 accuracy** | `% of cases where `rank=1` hypothesis points to the correct function` | ≥ 0.7 |
| **Top-3 accuracy** | `% of cases where correct function appears in top-3 hypotheses` | ≥ 0.9 |
| **Pattern-match precision** | Of pattern hits, how often is the pattern correct? | ≥ 0.85 |
| **Fix exactness** | Of proposed fixes, how many compile and pass the seeded test? | ≥ 0.5 |
| **Latency p50 / p95** | Time from `POST /analyze` to response | p95 < 30s |
| **Cost / case** | Claude API spend per analyzed case | reported, not enforced |

### 9.3 Runner

`tests/eval/runner.go`:
1. For each case, call `Engine.Analyze` against the real Claude API.
2. Compare predicted function to ground truth.
3. Run `go build` and `go test` of the patched sample-app.
4. Emit `report.md` and `report.json`.

### 9.4 Reporting

- `docs/EVAL.md` is updated after each Phase 1/2/3 gate.
- Numbers must be reproducible: re-running the eval gives the same table (modulo Claude's non-determinism; we set `temperature=0`).

---

## 10. Cost reality check (local-first)

With local deployment, infra cost ≈ **$0/month**. The only cost is the Claude API.

| Phase | Cases / run | Input tokens / case | Output tokens / case | Cost / case (Sonnet) | Cost / full eval |
|-------|-------------|---------------------|----------------------|----------------------|-------------------|
| Phase 1 eval (10 cases) | 10 | ~25k | ~3k | ~$0.12 | ~$1.20 |
| Phase 2 eval (20 cases) | 20 | ~25k | ~5k | ~$0.16 | ~$3.20 |
| Demo (live) | 5 | ~25k | ~3k | ~$0.12 | ~$0.60 |
| Buffer for iteration | — | — | — | — | **~$10 total** |

> Pricing based on Claude Sonnet: $3/M input, $15/M output. Subject to change.

Optimization levers (only if costs surprise us):
- Truncate `function.body` aggressively when not in the top-K code-linker candidates.
- Cache analysis results by `(codebase_hash, logs_hash)`.
- Use `claude-haiku-4` for the initial pattern pass and only escalate to Sonnet for hypothesis generation. (Stretch goal.)

---

## 11. What changes in the other docs

This is the cross-reference. **If you change SPECIFICATION.md, you must update the matching section of the other doc.**

| Section of this file | Mirrored in |
|---------------------|-------------|
| §2 Architecture | `ARCHITECTURE.md` |
| §3 Repository layout | `FOLDER_STRUCTURE.md` |
| §7 Roadmap | `PLAN.md` |
| §8 Local deployment | `INFRASTRUCTURE.md` |
| §4.5, §4.6, §4.7 API + storage | `DEVOPS.md` (CI/CD + monitoring only) |
| §10 Cost | `COST.md` |

---

## 12. Acceptance criteria (this is what "done" means)

The thesis is "done" when **all** of the following are true:

- [ ] `make build` succeeds on a clean checkout.
- [ ] `make test` passes (unit + integration + eval).
- [ ] `docker compose up` brings up the system end-to-end on a laptop.
- [ ] `helm install sre-agent ./configs/helm/sre-agent` works against a fresh `minikube start`.
- [ ] `POST /api/v1/analyze` against a seeded bug returns a `hypothesis` whose `suspect_function` matches ground truth for ≥ 70% of eval cases.
- [ ] For at least one seeded bug, the returned `fix.replacement` compiles and makes the test pass.
- [ ] `/metrics` exposes all metrics listed in §4.7.
- [ ] `docs/API.md`, `docs/EVAL.md`, `docs/DEPLOY.md` exist and are accurate.
- [ ] `README.md` has a 5-minute quickstart.
- [ ] Total Claude API spend across all development + evaluation is < $50.

---

*End of SPECIFICATION.md*
