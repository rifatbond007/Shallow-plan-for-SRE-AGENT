package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rifatbond007/sre-ai-agent/internal/codebase"
	"github.com/rifatbond007/sre-ai-agent/internal/ingest"
	"github.com/rifatbond007/sre-ai-agent/pkg/metrics"
)

type GeminiClient struct {
	apiKey  string
	model   string
	timeout time.Duration
}

func NewGeminiClient(apiKey, model string, timeout time.Duration) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
	}
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

func (g *GeminiClient) analyze(ctx context.Context, prompt, system string) (string, error) {
	start := time.Now()
	metrics.ClaudeRequestsTotal.Inc()

	sysPrompt := system
	if sysPrompt == "" {
		data, err := os.ReadFile("prompts/system.txt")
		if err != nil {
			sysPrompt = "You are an SRE assistant. Respond in JSON."
		} else {
			sysPrompt = string(data)
		}
	}

	fullPrompt := sysPrompt + "\n\n" + prompt

	req := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: fullPrompt}}},
		},
	}

	reqBody, _ := json.Marshal(req)

	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		metrics.ClaudeRequestDuration.Observe(time.Since(start).Seconds())
		return "", fmt.Errorf("request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		metrics.ClaudeRequestDuration.Observe(time.Since(start).Seconds())
		metrics.ClaudeErrorsTotal.WithLabelValues("network").Inc()
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	metrics.ClaudeRequestDuration.Observe(time.Since(start).Seconds())

	if resp.StatusCode != 200 {
		metrics.ClaudeErrorsTotal.WithLabelValues(fmt.Sprintf("http_%d", resp.StatusCode)).Inc()
		return "", fmt.Errorf("gemini API: %s", resp.Status)
	}

	var gr geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return gr.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiClient) AnalyzeIncident(ctx context.Context, inc ingest.Incident, candidates []codebase.ScoredFunction) ([]Hypothesis, error) {
	prompt, err := BuildPrompt(inc, candidates)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	text, err := g.analyze(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("gemini: %w", err)
	}

	text = cleanJSON(text)

	var resp claudeHypothesisResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var hypotheses []Hypothesis
	for _, h := range resp.Hypotheses {
		hyp := Hypothesis{
			ID:         fmt.Sprintf("hyp_%x", time.Now().UnixNano()),
			IncidentID: inc.ID,
			Rank:       h.Rank,
			Title:      h.Title,
			Summary:    h.Summary,
			Confidence: h.Confidence,
			LLMReasoning: h.Summary,
		}

		for _, logID := range h.EvidenceLogIDs {
			hyp.Evidence = append(hyp.Evidence, Evidence{
				Kind:       "log",
				LogEntryID: logID,
			})
		}

		if h.SuspectFunction != "" {
			for _, sf := range candidates {
				if sf.Function.ID == h.SuspectFunction {
					hyp.SuspectCode = CodeRef{
						File: sf.Function.File,
						Line: sf.Function.Line,
					}
					break
				}
			}
		}

		hypotheses = append(hypotheses, hyp)
	}

	if len(hypotheses) == 0 {
		return nil, fmt.Errorf("no hypotheses returned")
	}

	return hypotheses, nil
}

func (g *GeminiClient) GenerateFix(ctx context.Context, hypothesis Hypothesis, candidates []codebase.ScoredFunction) (Fix, error) {
	prompt, err := BuildFixPrompt(hypothesis, candidates)
	if err != nil {
		return Fix{}, fmt.Errorf("build fix prompt: %w", err)
	}

	text, err := g.analyze(ctx, prompt, "")
	if err != nil {
		return Fix{}, fmt.Errorf("gemini: %w", err)
	}

	text = cleanJSON(text)

	var resp claudeFixResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return Fix{}, fmt.Errorf("parse fix: %w", err)
	}

	return Fix{
		Summary:     resp.Summary,
		UnifiedDiff: resp.UnifiedDiff,
		Replacement: resp.Replacement,
		Confidence:  hypothesis.Confidence,
		Caveats:     resp.Caveats,
	}, nil
}
