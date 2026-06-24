package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rifatbond007/sre-ai-agent/internal/analysis"
)

type EvalCase struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	LogsPath     string `json:"logs_path"`
	CodebasePath string `json:"codebase_path"`
	GroundTruth  struct {
		Function   string `json:"function"`
		FixSummary string `json:"fix_summary"`
	} `json:"ground_truth"`
}

type EvalResult struct {
	CaseID        string  `json:"case_id"`
	Name          string  `json:"name"`
	Top1Hit       bool    `json:"top1_hit"`
	Top3Hit       bool    `json:"top3_hit"`
	Top1Func      string  `json:"top1_func"`
	GroundTruth   string  `json:"ground_truth"`
	Top1Confidence float64 `json:"top1_confidence"`
	DurationMS    int64   `json:"duration_ms"`
}

type EvalReport struct {
	Results      []EvalResult `json:"results"`
	Top1Accuracy float64      `json:"top1_accuracy"`
	Top3Accuracy float64      `json:"top3_accuracy"`
	TotalCases   int          `json:"total_cases"`
	PassedTop1   int          `json:"passed_top1"`
	PassedTop3   int          `json:"passed_top3"`
}

func main() {
	data, err := os.ReadFile("tests/eval/cases.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read cases: %v\n", err)
		os.Exit(1)
	}

	var cases []EvalCase
	if err := json.Unmarshal(data, &cases); err != nil {
		fmt.Fprintf(os.Stderr, "parse cases: %v\n", err)
		os.Exit(1)
	}

	claude := analysis.NewClaudeClient(
		os.Getenv("SRE_AGENT_ANTHROPIC_API_KEY"),
		"claude-sonnet-4-20250514",
		60*time.Second,
	)

	eng := analysis.NewEngine(claude, 5_000_000)

	var results []EvalResult
	for _, c := range cases {
		start := time.Now()

		logData, err := os.ReadFile(c.LogsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read logs %s: %v\n", c.LogsPath, err)
			continue
		}

		codebasePath, _ := filepath.Abs(c.CodebasePath)

		result, err := eng.Analyze(context.Background(), analysis.AnalyzeRequest{
			Logs:         string(logData),
			CodebasePath: codebasePath,
			TopK:         3,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "analyze %s: %v\n", c.ID, err)
			continue
		}

		top1Hit := false
		top3Hit := false
		top1Func := ""

		for i, h := range result.Hypotheses {
			funcName := h.SuspectCode.File
			if i == 0 {
				top1Func = funcName
			}
			if funcName == c.GroundTruth.Function {
				if i == 0 {
					top1Hit = true
				}
				if i < 3 {
					top3Hit = true
				}
			}
		}

		er := EvalResult{
			CaseID:        c.ID,
			Name:          c.Name,
			Top1Hit:       top1Hit,
			Top3Hit:       top3Hit,
			Top1Func:      top1Func,
			GroundTruth:   c.GroundTruth.Function,
			DurationMS:    time.Since(start).Milliseconds(),
		}
		if len(result.Hypotheses) > 0 {
			er.Top1Confidence = result.Hypotheses[0].Confidence
		}

		results = append(results, er)

		status := "PASS"
		if !top1Hit {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s: top1=%v top3=%v (%.2f, %dms)\n",
			status, c.Name, top1Hit, top3Hit, er.Top1Confidence, er.DurationMS)
	}

	passedTop1 := 0
	passedTop3 := 0
	for _, r := range results {
		if r.Top1Hit {
			passedTop1++
		}
		if r.Top3Hit {
			passedTop3++
		}
	}

	report := EvalReport{
		Results:      results,
		TotalCases:   len(results),
		PassedTop1:   passedTop1,
		PassedTop3:   passedTop3,
		Top1Accuracy: float64(passedTop1) / float64(len(results)),
		Top3Accuracy: float64(passedTop3) / float64(len(results)),
	}

	reportData, _ := json.MarshalIndent(report, "", "  ")
	os.WriteFile("tests/eval/report.json", reportData, 0644)

	fmt.Printf("\n=== EVAL REPORT ===\n")
	fmt.Printf("Top-1:  %d/%d (%.1f%%)\n", passedTop1, len(results), report.Top1Accuracy*100)
	fmt.Printf("Top-3:  %d/%d (%.1f%%)\n", passedTop3, len(results), report.Top3Accuracy*100)
	fmt.Printf("Report: tests/eval/report.json\n")
}
