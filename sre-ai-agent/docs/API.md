# SRE AI Agent — API Reference

Base URL: `http://localhost:8080/api/v1`

## Health & Readiness

### `GET /api/v1/healthz`

Returns 200 when the server is alive. No auth required.

```json
{"status": "ok"}
```

### `GET /api/v1/readyz`

Returns 200 when the server is ready to accept traffic.

```json
{"status": "ok"}
```

---

## Analysis

### `POST /api/v1/analyze`

Analyze logs against a codebase and return ranked hypotheses + fixes.

**Request body:**

```json
{
  "logs": "raw log text, one entry per line",
  "codebase_path": "/path/to/go/code",
  "top_k": 3
}
```

| Field | Type | Default | Description |
|---|---|---|---|
| `logs` | string | — | Raw log lines (supports nginx access, nginx error, JSON app logs) |
| `codebase_path` | string | `/codebase` | Directory of Go source to analyze |
| `top_k` | int | `3` | Number of candidate functions to consider |

**Response (200):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-06-25T10:00:00Z",
  "incidents": [
    {
      "ID": "inc_184c40cc3829",
      "StartedAt": "2025-06-25T10:00:01Z",
      "EndedAt": "2025-06-25T10:00:03Z",
      "Entries": [
        {
          "ID": "8e958d38117acc3e",
          "Timestamp": "2025-06-25T10:00:01Z",
          "Source": "nginx_error",
          "Severity": "ERROR",
          "Message": "upstream timed out...",
          "Fields": {"cid": "1", "pid_tid": "1234#1234"},
          "Raw": "2025/06/25 10:00:01 [error] ...",
          "StackTrace": ""
        }
      ],
      "Signatures": ["nginx_504"],
      "TopError": "upstream timed out...",
      "ErrorCount": 3,
      "WarnCount": 0
    }
  ],
  "hypotheses": [
    {
      "id": "hyp_...",
      "incident_id": "inc_184c40cc3829",
      "rank": 1,
      "title": "Missing HTTP timeout in payment call",
      "summary": "checkoutHandler calls paymentService without...",
      "confidence": 0.85,
      "evidence": [
        {"kind": "log", "log_entry_id": "8e958d38117acc3e", "description": "upstream timeout"}
      ],
      "suspect_code": {"file": "main.go", "line": 42, "snippet": "resp, err := http.Post(...)"},
      "pattern_hit": {"pattern_id": "nginx_504", "label": "Upstream timeout"}
    }
  ],
  "fixes": [
    {
      "hypothesis_id": "hyp_...",
      "summary": "Add 5s timeout to HTTP client",
      "unified_diff": "--- a/main.go\n+++ b/main.go\n@@ -40,3 +40,3 @@\n-http.DefaultClient\n+&http.Client{Timeout: 5 * time.Second}",
      "confidence": 0.85,
      "caveats": ["Assumes payment service SLA < 5s"]
    }
  ],
  "summary": "1 incidents, 3 hypotheses, 1 fix",
  "duration_ms": 2350
}
```

**Error responses:**

| Status | Code | Description |
|---|---|---|
| 400 | `INVALID_REQUEST` | Missing or malformed JSON body |
| 400 | `INVALID_REQUEST` | `logs` field is empty |
| 413 | `LOGS_TOO_LARGE` | Request body exceeds `SRE_AGENT_MAX_LOG_BYTES` |
| 502 | `CLAUDE_UPSTREAM` | LLM provider returned an error |
| 504 | `TIMEOUT` | LLM request timed out |

---

### `GET /api/v1/analyze/ws`

WebSocket endpoint for real-time streaming analysis.

**Protocol:**

1. Client connects via WebSocket to `ws://host:8080/api/v1/analyze/ws`
2. Client sends a single JSON message matching the `POST /api/v1/analyze` request body
3. Server streams progress events as JSON messages:

| Type | Fields | Description |
|---|---|---|
| `progress` | `stage`, `pct` | Pipeline stage update |
| `incident` | `data` (Incident) | Incident detected |
| `hypothesis` | `data` (Hypothesis) | Hypothesis generated |
| `fix` | `data` (Fix) | Fix generated |
| `error` | `error` | Pipeline error |
| `done` | `data` (result ID) | Stream complete |
| `result` | `data` (AnalysisResult) | Full result JSON |

4. Server closes the connection after sending `result` or `error`

---

### `GET /api/v1/hypotheses/:id`

Retrieve a previously completed analysis by its result ID.

| Parameter | Description |
|---|---|
| `id` | Result UUID returned by `POST /api/v1/analyze` |

**Response:** Same schema as `POST /api/v1/analyze` response. Returns `404` if the result has expired from cache.

---

## Metrics

### `GET /metrics`

Prometheus metrics endpoint. Exposes:

- `sre_agent_http_requests_total` — counter by method, path, status
- `sre_agent_http_request_duration_seconds` — histogram by method, path
- `sre_agent_llm_requests_total` — counter
- `sre_agent_llm_request_duration_seconds` — histogram
- `sre_agent_hypotheses_generated_total` — counter
- `sre_agent_cache_hits_total` / `sre_agent_cache_misses_total` — counters
- `sre_agent_llm_api_errors_total` — counter by error kind

---

## Authentication (optional)

Set `SRE_AGENT_API_KEY` in the environment. When set, all endpoints except `healthz`, `readyz`, and `/metrics` require:

```
X-API-Key: <your-api-key>
```

Or as a query parameter: `?api_key=<your-api-key>`

---

## Configuration Reference

| Env Var | Default | Description |
|---|---|---|
| `SRE_AGENT_PORT` | `8080` | HTTP listen port |
| `SRE_AGENT_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `SRE_AGENT_LLM_PROVIDER` | `claude` | `claude` or `gemini` |
| `SRE_AGENT_ANTHROPIC_API_KEY` | — | Claude API key |
| `SRE_AGENT_ANTHROPIC_MODEL` | `claude-sonnet-4-20250514` | Claude model ID |
| `SRE_AGENT_ANTHROPIC_TIMEOUT` | `30s` | Claude request timeout |
| `SRE_AGENT_GEMINI_API_KEY` | — | Gemini API key |
| `SRE_AGENT_GEMINI_MODEL` | `gemini-2.0-flash` | Gemini model ID |
| `SRE_AGENT_GEMINI_TIMEOUT` | `30s` | Gemini request timeout |
| `SRE_AGENT_CODEBASE_PATH` | `/codebase` | Path to Go source code |
| `SRE_AGENT_MAX_LOG_BYTES` | `5000000` | Max log input size |
| `SRE_AGENT_INCIDENT_WINDOW` | `5m` | Time window for grouping incidents |
| `SRE_AGENT_INCIDENT_MIN_SIZE` | `3` | Min log lines to form an incident |
| `SRE_AGENT_HYPOTHESIS_TOP_K` | `3` | Candidate functions per incident |
| `SRE_AGENT_CACHE_TTL` | `1h` | Result cache TTL |
| `SRE_AGENT_CACHE_MAX_ENTRIES` | `512` | Max cached results |
| `SRE_AGENT_RATE_LIMIT_RPS` | `5` | Rate limiter requests/sec |
| `SRE_AGENT_RATE_LIMIT_BURST` | `10` | Rate limiter burst |
| `SRE_AGENT_API_KEY` | — | Optional static API key |
