package codebase

import (
	"sort"
	"strings"

	"github.com/rifatbond007/sre-ai-agent/internal/ingest"
)

type FunctionLinker struct {
	idx *Index
	cg  *CallGraph
}

func NewLinker(idx *Index) *FunctionLinker {
	return &FunctionLinker{
		idx: idx,
		cg:  NewCallGraph(idx),
	}
}

func (l *FunctionLinker) CandidateFunctions(inc ingest.Incident, k int) []ScoredFunction {
	logText := extractLogText(inc)
	logLower := strings.ToLower(logText)

	var scored []ScoredFunction
	for _, fn := range l.idx.Functions {
		s := l.score(fn, logLower)
		if s > 0 {
			scored = append(scored, ScoredFunction{
				Function: fn,
				Score:    s,
				Reasons:  l.reasons(fn, logLower),
			})
		}
	}

	// Boost functions reachable from top scorers
	topIDs := make(map[string]bool)
	for _, sf := range scored {
		topIDs[sf.Function.ID] = true
	}
	for _, sf := range scored {
		if sf.Score >= 4 {
			neighbors := l.cg.BFS(sf.Function.ID, 2)
			for _, n := range neighbors {
				if !topIDs[n.ID] {
					scored = append(scored, ScoredFunction{
						Function: n,
						Score:    sf.Score * 0.6,
						Reasons:  []string{"reachable from " + sf.Function.ID},
					})
					topIDs[n.ID] = true
				}
			}
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if len(scored) > k {
		scored = scored[:k]
	}
	return scored
}

func (l *FunctionLinker) score(fn Function, logLower string) float64 {
	var s float64

	nameLower := strings.ToLower(fn.Name)
	if strings.Contains(logLower, nameLower) {
		s += 5
	}

	docLower := strings.ToLower(fn.Doc)
	if docLower != "" {
		words := strings.Fields(docLower)
		for _, w := range words {
			if len(w) > 3 && strings.Contains(logLower, w) {
				s += 4
				break
			}
		}
	}

	if strings.Contains(logLower, "error") && strings.Contains(fn.Signature, "error") {
		s += 3
	}

	if strings.Contains(fn.Body, "nil") && strings.Contains(logLower, "nil") {
		s += 1
	}
	if strings.Contains(fn.Body, "timeout") && strings.Contains(logLower, "timeout") {
		s += 1
	}
	if strings.Contains(fn.Body, "context") && strings.Contains(logLower, "deadline") {
		s += 1
	}

	return s
}

func (l *FunctionLinker) reasons(fn Function, logLower string) []string {
	var r []string

	nameLower := strings.ToLower(fn.Name)
	if strings.Contains(logLower, nameLower) {
		r = append(r, "function name matches log message")
	}

	docLower := strings.ToLower(fn.Doc)
	if docLower != "" {
		words := strings.Fields(docLower)
		for _, w := range words {
			if len(w) > 3 && strings.Contains(logLower, w) {
				r = append(r, "doc keyword matches log message")
				break
			}
		}
	}

	if strings.Contains(fn.Signature, "error") && strings.Contains(logLower, "error") {
		r = append(r, "function returns/handles error type")
	}

	return r
}

func extractLogText(inc ingest.Incident) string {
	var b strings.Builder
	for _, e := range inc.Entries {
		b.WriteString(e.Message)
		b.WriteString(" ")
		for _, v := range e.Fields {
			b.WriteString(v)
			b.WriteString(" ")
		}
	}
	return b.String()
}
