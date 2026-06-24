package analysis

import (
	"context"
	"time"

	"github.com/rifatbond007/sre-ai-agent/internal/ingest"
)

type Hypothesis struct {
	ID           string      `json:"id"`
	IncidentID   string      `json:"incident_id"`
	Rank         int         `json:"rank"`
	Title        string      `json:"title"`
	Summary      string      `json:"summary"`
	Confidence   float64     `json:"confidence"`
	Evidence     []Evidence  `json:"evidence"`
	SuspectCode  CodeRef     `json:"suspect_code"`
	RelatedFuncs []CodeRef   `json:"related_funcs,omitempty"`
	PatternHit   *PatternMatch `json:"pattern_hit,omitempty"`
	LLMReasoning string      `json:"llm_reasoning,omitempty"`
}

type Evidence struct {
	Kind        string   `json:"kind"`
	LogEntryID  string   `json:"log_entry_id,omitempty"`
	CodeRef     *CodeRef `json:"code_ref,omitempty"`
	Description string   `json:"description"`
}

type CodeRef struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet,omitempty"`
}

type Fix struct {
	HypothesisID string   `json:"hypothesis_id"`
	Summary      string   `json:"summary"`
	UnifiedDiff  string   `json:"unified_diff,omitempty"`
	Replacement  string   `json:"replacement,omitempty"`
	Confidence   float64  `json:"confidence"`
	Caveats      []string `json:"caveats,omitempty"`
}

type AnalysisResult struct {
	ID         string            `json:"id"`
	CreatedAt  time.Time         `json:"created_at"`
	Incidents  []ingest.Incident `json:"incidents"`
	Hypotheses []Hypothesis      `json:"hypotheses"`
	Fixes      []Fix             `json:"fixes,omitempty"`
	Summary    string            `json:"summary"`
	DurationMS int64             `json:"duration_ms"`
}

type ProgressEvent struct {
	Type  string `json:"type"`
	Stage string `json:"stage,omitempty"`
	Pct   int    `json:"pct,omitempty"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type AnalyzeRequest struct {
	Logs         string
	CodebasePath string
	TopK         int
}

type Engine interface {
	Analyze(ctx context.Context, req AnalyzeRequest) (*AnalysisResult, error)
	AnalyzeStream(ctx context.Context, req AnalyzeRequest, sink func(ProgressEvent) error) (*AnalysisResult, error)
}

type Pattern struct {
	ID       string
	Severity ingest.Severity
	Regex    string
	Label    string
}

type PatternMatch struct {
	PatternID string
	Label     string
	Severity  ingest.Severity
	Lines     []string
}
