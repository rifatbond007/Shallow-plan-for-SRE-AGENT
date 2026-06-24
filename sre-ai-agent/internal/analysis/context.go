package analysis

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/rifatbond007/sre-ai-agent/internal/codebase"
	"github.com/rifatbond007/sre-ai-agent/internal/ingest"
)

func BuildPrompt(inc ingest.Incident, candidates []codebase.ScoredFunction) (string, error) {
	tmpl, err := template.New("hypothesis").Parse(hypothesisPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("template: %w", err)
	}

	formattedLogs := formatLogEntries(inc.Entries)
	formattedFuncs := formatFunctions(candidates)

	data := map[string]any{
		"incident": map[string]any{
			"id":              inc.ID,
			"started_at":      inc.StartedAt.Format(time.RFC3339),
			"ended_at":        inc.EndedAt.Format(time.RFC3339),
			"error_count":     inc.ErrorCount,
			"warn_count":      inc.WarnCount,
			"top_error":       inc.TopError,
			"signatures":      strings.Join(inc.Signatures, ", "),
		},
		"formatted_log_entries": formattedLogs,
		"functions":             formattedFuncs,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	return buf.String(), nil
}

func formatLogEntries(entries []ingest.LogEntry) string {
	limit := 20
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	var b strings.Builder
	for i, e := range entries {
		b.WriteString(fmt.Sprintf("[%d] %s | %s | %s | %s\n",
			i+1,
			e.Timestamp.Format("15:04:05"),
			e.Source,
			e.Severity,
			e.Message,
		))
	}
	return b.String()
}

func formatFunctions(candidates []codebase.ScoredFunction) string {
	var b strings.Builder
	for _, sf := range candidates {
		b.WriteString(fmt.Sprintf("  - %s  (%s:%d)\n", sf.Function.ID, sf.Function.File, sf.Function.Line))
		b.WriteString(fmt.Sprintf("  Signature: %s\n", sf.Function.Signature))
		b.WriteString("  Body:\n  ```go\n")
		body := sf.Function.Body
		if len(body) > 3000 {
			body = body[:3000] + "\n  // ... truncated"
		}
		for _, line := range strings.Split(body, "\n") {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("  ```\n")
	}
	return b.String()
}

func BuildFixPrompt(hypothesis Hypothesis, candidates []codebase.ScoredFunction) (string, error) {
	var targetFunc string
	var targetFile string
	var targetLine int
	var targetBody string

	for _, sf := range candidates {
		if sf.Function.ID == hypothesis.SuspectFunction {
			targetFunc = sf.Function.ID
			targetFile = sf.Function.File
			targetLine = sf.Function.Line
			targetBody = sf.Function.Body
			break
		}
	}

	if targetFunc == "" && len(candidates) > 0 {
		targetFunc = candidates[0].Function.ID
		targetFile = candidates[0].Function.File
		targetLine = candidates[0].Function.Line
		targetBody = candidates[0].Function.Body
	}

	return fmt.Sprintf(fixPromptTemplate, hypothesis.ID, hypothesis.Title, hypothesis.Summary, hypothesis.Confidence, targetFunc, targetFile, targetLine, targetBody), nil
}

const hypothesisPromptTemplate = `INCIDENT #{{.incident.id}}
Window: {{.incident.started_at}} -> {{.incident.ended_at}}
Errors: {{.incident.error_count}}, Warnings: {{.incident.warn_count}}
Top message: {{.incident.top_error}}
Pattern matches: {{.incident.signatures}}

LOG ENTRIES (most recent first):
{{.formatted_log_entries}}

CANDIDATE FUNCTIONS (ranked by code-linker score):
{{.functions}}
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
}`

const fixPromptTemplate = `Top hypothesis:
ID: %s
Title: %s
Summary: %s
Confidence: %.2f

Function to patch: %s at %s:%d
Current body:
` + "```go\n%s\n```\n" + `
Produce a JSON object:
{
  "summary": "...",
  "replacement": "<full new function body, valid Go>",
  "unified_diff": "<optional unified diff>",
  "caveats": ["..."]
}`
