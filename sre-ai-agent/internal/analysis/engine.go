package analysis

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rifatbond007/sre-ai-agent/internal/codebase"
	"github.com/rifatbond007/sre-ai-agent/internal/ingest"
	"github.com/rifatbond007/sre-ai-agent/pkg/metrics"
)

type engine struct {
	multiParser  *ingest.MultiParser
	grouper      ingest.Grouper
	patternMatch *PatternMatcher
	llm          LLMClient
	ranker       *Ranker
	maxLogBytes  int
}

func NewEngine(llm LLMClient, maxLogBytes int) Engine {
	return &engine{
		multiParser:  ingest.NewMultiParser(),
		grouper:      ingest.DefaultGrouper(),
		patternMatch: NewPatternMatcher(),
		llm:          llm,
		ranker:       NewRanker(),
		maxLogBytes:  maxLogBytes,
	}
}

func (e *engine) Analyze(ctx context.Context, req AnalyzeRequest) (*AnalysisResult, error) {
	start := time.Now()

	if len(req.Logs) > e.maxLogBytes {
		return nil, fmt.Errorf("logs exceed max size of %d bytes", e.maxLogBytes)
	}

	logs, err := e.parseLogs(req.Logs)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	incidents := e.groupEntries(logs)

	idx, err := codebase.BuildIndex(req.CodebasePath)
	if err != nil {
		return nil, fmt.Errorf("codebase: %w", err)
	}
	linker := codebase.NewLinker(idx)

	var allHypotheses []Hypothesis
	var allFixes []Fix

	for _, inc := range incidents {
		pattern := e.patternMatch.Match(inc)

		candidates := linker.CandidateFunctions(inc, req.TopK)

		hypotheses, err := e.llm.AnalyzeIncident(ctx, inc, candidates)
		if err != nil {
			log.Printf("LLM AnalyzeIncident error: %v", err)
			continue
		}

		e.ranker.Rank(hypotheses, pattern)

		fix, err := e.llm.GenerateFix(ctx, hypotheses[0], candidates)
		if err == nil {
			fix.HypothesisID = hypotheses[0].ID
			allFixes = append(allFixes, fix)
		}

		allHypotheses = append(allHypotheses, hypotheses...)
	}

	metrics.HypothesesGenerated.Add(float64(len(allHypotheses)))

	duration := time.Since(start)

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("%d incidents, %d hypotheses", len(incidents), len(allHypotheses)))
	if len(allFixes) > 0 {
		summary.WriteString(fmt.Sprintf(", %d fixes", len(allFixes)))
	}

	return &AnalysisResult{
		ID:         fmt.Sprintf("res_%x", time.Now().UnixNano()),
		CreatedAt:  time.Now().UTC(),
		Incidents:  incidents,
		Hypotheses: allHypotheses,
		Fixes:      allFixes,
		Summary:    summary.String(),
		DurationMS: duration.Milliseconds(),
	}, nil
}

func (e *engine) AnalyzeStream(ctx context.Context, req AnalyzeRequest, sink func(ProgressEvent) error) (*AnalysisResult, error) {
	start := time.Now()

	sink(ProgressEvent{Type: "progress", Stage: "parsing", Pct: 10})

	logs, err := e.parseLogs(req.Logs)
	if err != nil {
		sink(ProgressEvent{Type: "error", Error: err.Error()})
		return nil, err
	}

	sink(ProgressEvent{Type: "progress", Stage: "grouping", Pct: 20})
	incidents := e.groupEntries(logs)

	for _, inc := range incidents {
		sink(ProgressEvent{Type: "incident", Data: inc})
	}

	sink(ProgressEvent{Type: "progress", Stage: "codebase", Pct: 40})
	idx, err := codebase.BuildIndex(req.CodebasePath)
	if err != nil {
		sink(ProgressEvent{Type: "error", Error: err.Error()})
		return nil, err
	}
	linker := codebase.NewLinker(idx)

	sink(ProgressEvent{Type: "progress", Stage: "analysis", Pct: 60})

	var allHypotheses []Hypothesis
	var allFixes []Fix

	for i, inc := range incidents {
		pattern := e.patternMatch.Match(inc)
		candidates := linker.CandidateFunctions(inc, req.TopK)

		hypotheses, err := e.llm.AnalyzeIncident(ctx, inc, candidates)
		if err != nil {
			log.Printf("LLM AnalyzeIncident error: %v", err)
			continue
		}

		e.ranker.Rank(hypotheses, pattern)

		for _, h := range hypotheses {
			sink(ProgressEvent{Type: "hypothesis", Data: h})
		}

		fix, err := e.llm.GenerateFix(ctx, hypotheses[0], candidates)
		if err == nil {
			fix.HypothesisID = hypotheses[0].ID
			allFixes = append(allFixes, fix)
			sink(ProgressEvent{Type: "fix", Data: fix})
		}

		allHypotheses = append(allHypotheses, hypotheses...)

		pct := 60 + ((i + 1) * 30 / len(incidents))
		sink(ProgressEvent{Type: "progress", Stage: "analysis", Pct: pct})
	}

	metrics.HypothesesGenerated.Add(float64(len(allHypotheses)))

	duration := time.Since(start)

	result := &AnalysisResult{
		ID:         fmt.Sprintf("res_%x", time.Now().UnixNano()),
		CreatedAt:  time.Now().UTC(),
		Incidents:  incidents,
		Hypotheses: allHypotheses,
		Fixes:      allFixes,
		DurationMS: duration.Milliseconds(),
	}

	sink(ProgressEvent{Type: "done", Data: result.ID})
	return result, nil
}

func (e *engine) parseLogs(raw string) ([]ingest.LogEntry, error) {
	r := strings.NewReader(raw)
	entries, err := e.multiParser.ParseAll(r)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (e *engine) groupEntries(entries []ingest.LogEntry) []ingest.Incident {
	if len(entries) == 0 {
		return nil
	}
	return e.grouper.Group(entries)
}
