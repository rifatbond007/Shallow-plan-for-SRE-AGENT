# SRE AI Agent - Cost Breakdown (v2)

> **Source of truth:** [`SPECIFICATION.md`](../SPECIFICATION.md), §10.
> This is a thesis project that runs on a laptop. There is **no
> infrastructure bill** — only the Claude API bill, which is small.

---

## 1. TL;DR

| Category | Cost (USD) |
|----------|-----------|
| Compute (your laptop) | $0 |
| Storage (your laptop) | $0 |
| Networking (your laptop) | $0 |
| Kubernetes (Minikube/Kind, local) | $0 |
| Prometheus / Grafana (local containers) | $0 |
| **Claude API (Anthropic)** | **~$10–30 total across the whole thesis** |

> **Total infra: $0/month. Total API across 4 months: $10–30.**

---

## 2. Why this is so much smaller than v1

The previous cost doc budgeted for AWS EKS + RDS + ElastiCache + CloudWatch
+ WAF + Route 53 ≈ $510/month. **For a thesis demo run on a laptop, none
of that is necessary.** All those services are replaced by `docker compose`
on a developer's machine, costing exactly nothing.

The only variable cost is the Claude API. Below is a realistic estimate.

---

## 3. Claude API pricing (Sonnet 4.6, as of 2026-05)

| Direction | Price per 1M tokens |
|-----------|---------------------|
| Input | $3.00 |
| Output | $15.00 |

> Confirm current pricing at https://www.anthropic.com/pricing before
> submitting. If Anthropic has changed prices, update this table.

### Haiku (optional fast-path)

| Direction | Price per 1M tokens |
|-----------|---------------------|
| Input | $0.25 |
| Output | $1.25 |

We default to Sonnet for quality. Haiku is a stretch goal for the
low-severity filter.

---

## 4. Per-case token estimate

A typical `POST /api/v1/analyze` request sends:

- System prompt: ~300 tokens
- Incident block (≤ 20 log lines + 3 candidate functions, bodies
  truncated): ~20,000 tokens input
- Hypothesis response: ~1,500 tokens output
- Fix response (if requested): ~2,000 tokens output

So one full analysis (hypothesis + fix):

- **Input:**  ~20,000 tokens  → $0.06
- **Output:** ~3,500 tokens   → $0.05
- **Per case:** **~$0.11**

A hypothesis-only call (Phase 1):

- **Input:**  ~20,000 tokens → $0.06
- **Output:** ~1,500 tokens  → $0.02
- **Per case:** **~$0.08**

---

## 5. Realistic spend across the thesis

| Activity | Cases | Cost/case | Subtotal |
|----------|------:|----------:|---------:|
| Dev iteration (manual curl) | 100 | $0.08 | $8 |
| Phase 1 eval (10 cases × 5 runs) | 50 | $0.08 | $4 |
| Phase 2 eval (20 cases × 5 runs) | 100 | $0.11 | $11 |
| Phase 3 eval (20 cases × 3 runs) | 60 | $0.11 | $7 |
| Demo runs (live, ~10 audiences) | 50 | $0.11 | $6 |
| Buffer (reruns, prompt tuning) | — | — | $14 |
| **Total** | | | **~$50** |

This matches the acceptance criterion in SPECIFICATION.md §12:
"Total Claude API spend across all development + evaluation is < $50 USD."

If we land closer to $30, even better. We will not exceed $50 unless
something is fundamentally wrong (e.g., we forgot to truncate code bodies).

---

## 6. Cost-control levers

Already implemented in SPECIFICATION.md §4.4:

1. **Truncate `function.body` to fit a token budget.** Default ~30k tokens
   of combined context. Drop lowest-scored functions first.
2. **Cache results by `(codebase_hash, logs_hash)`.** A repeated request
   is free.
3. **Cap `SRE_AGENT_ANTHROPIC_MAX_TOKENS` per call.** Default 2048 for
   hypothesis, 2048 for fix.
4. **Cap `SRE_AGENT_MAX_LOG_BYTES`.** Default 5 MB — limits log spam.
5. **Rate-limit per IP.** Default 5 RPS, burst 10.

If costs still surprise us:

6. Run the deterministic pattern matcher first. If a high-confidence
   pattern hits, **skip the LLM call** and return the canned hypothesis.
   (Stretch goal — Phase 2+.)
7. Use Haiku for the "is this an incident at all?" pre-filter.
   (Stretch goal.)
8. Batch multiple log entries into one LLM call. Already do this for
   an incident; ensure we don't accidentally re-call per log line.

---

## 7. What we are NOT paying for

| Service | v1 cost | v2 cost |
|---------|--------:|--------:|
| AWS EKS (control plane) | $75/mo | $0 |
| AWS EC2 worker nodes | $180/mo | $0 |
| AWS RDS PostgreSQL | $80/mo | $0 |
| AWS Application Load Balancer | $25/mo | $0 |
| AWS CloudWatch Logs | $45/mo | $0 |
| AWS CloudWatch Metrics | $15/mo | $0 |
| AWS NAT Gateway | $35/mo | $0 |
| AWS S3 (backup) | $20/mo | $0 |
| AWS Route 53 | $5/mo | $0 |
| AWS Secrets Manager | $5/mo | $0 |
| Pinecone / Weaviate (vector DB) | $50–200/mo | $0 |
| Datadog (full stack) | $200/mo | $0 |
| **Total monthly** | **~$510** | **$0** |

That is roughly **$6,000/year saved** by being honest about scope.

---

## 8. Reporting

After the thesis submission, attach a copy of your actual Anthropic
dashboard showing total spend. The expected line is "Developer API: $X.XX
over Y months." If it matches the table in §5 within 50%, the cost
discipline worked.

---

## 9. What this document does NOT cover

- Cloud cost projections — there are no cloud resources
- Reserved-instance discounts — not relevant
- Spot instance savings — not relevant
- Enterprise support contracts — not relevant
- Per-region pricing variations — not relevant

If the thesis examiner asks "what would this cost in production?", the
honest answer is: "I haven't priced that because it's out of scope. The
specification (§1.2) treats cloud deployment as future work."
