package codebase

import (
	"time"

	"github.com/rifatbond007/sre-ai-agent/internal/ingest"
)

type Function struct {
	ID         string
	PkgPath    string
	Name       string
	Receiver   string
	File       string
	Line       int
	EndLine    int
	Signature  string
	Body       string
	Calls      []string
	IsExported bool
	Doc        string
}

type Index struct {
	Functions map[string]Function
	ByFile    map[string][]string
	Roots     []string
	Built     time.Time
}

type ScoredFunction struct {
	Function Function
	Score    float64
	Reasons  []string
}

type Linker interface {
	CandidateFunctions(inc ingest.Incident, k int) []ScoredFunction
}
