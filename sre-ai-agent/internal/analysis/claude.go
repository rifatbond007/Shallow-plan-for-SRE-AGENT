package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rifatbond007/sre-ai-agent/internal/codebase"
	"github.com/rifatbond007/sre-ai-agent/internal/ingest"
	"github.com/rifatbond007/sre-ai-agent/pkg/metrics"
)

type ClaudeClient struct {
	apiKey  string
	model   string
	timeout time.Duration
}

func NewClaudeClient(apiKey, model string, timeout time.Duration) *ClaudeClient {
	return &ClaudeClient{
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
	}
}

type claudeHypothesisResponse struct {
	Hypotheses []struct {
		Rank             int      `json:"rank"`
		Title            string   `json:"title"`
		Summary          string   `json:"summary"`
		Confidence       float64  `json:"confidence"`
		SuspectFunction  string   `json:"suspect_function"`
		EvidenceLogIDs   []string `json:"evidence_log_ids"`
	} `json:"hypotheses"`
}

type claudeFixResponse struct {
	Summary     string   `json:"summary"`
	Replacement string   `json:"replacement"`
	UnifiedDiff string   `json:"unified_diff"`
	Caveats     []string `json:"caveats"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (c *ClaudeClient) analyze(ctx context.Context, prompt, system string) (string, error) {
	start := time.Now()
	metrics.LLMRequestsTotal.Inc()

	sysPrompt := system
	if sysPrompt == "" {
		data, err := os.ReadFile("prompts/system.txt")
		if err != nil {
			sysPrompt = "You are an SRE assistant. Respond in JSON."
		} else {
			sysPrompt = string(data)
		}
	}

	req := claudeRequest{
		Model:     c.model,
		MaxTokens: 2048,
		System:    sysPrompt,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	reqBody, _ := json.Marshal(req)

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		metrics.LLMRequestDuration.Observe(time.Since(start).Seconds())
		return "", fmt.Errorf("request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		metrics.LLMRequestDuration.Observe(time.Since(start).Seconds())
		metrics.LLMErrorsTotal.WithLabelValues("network").Inc()
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	metrics.LLMRequestDuration.Observe(time.Since(start).Seconds())

	if resp.StatusCode != 200 {
		metrics.LLMErrorsTotal.WithLabelValues(fmt.Sprintf("http_%d", resp.StatusCode)).Inc()
		return "", fmt.Errorf("claude API: %s", resp.Status)
	}

	var cr claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	if len(cr.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return cr.Content[0].Text, nil
}

func (c *ClaudeClient) AnalyzeIncident(ctx context.Context, inc ingest.Incident, candidates []codebase.ScoredFunction) ([]Hypothesis, error) {
	prompt, err := BuildPrompt(inc, candidates)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	text, err := c.analyze(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}

	text = cleanJSON(text)

	var resp claudeHypothesisResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var hypotheses []Hypothesis
	for _, h := range resp.Hypotheses {
		hyp := Hypothesis{
			ID:              fmt.Sprintf("hyp_%x", time.Now().UnixNano()),
			IncidentID:      inc.ID,
			Rank:            h.Rank,
			Title:           h.Title,
			Summary:         h.Summary,
			Confidence:      h.Confidence,
			SuspectFunction: h.SuspectFunction,
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

func (c *ClaudeClient) GenerateFix(ctx context.Context, hypothesis Hypothesis, candidates []codebase.ScoredFunction) (Fix, error) {
	prompt, err := BuildFixPrompt(hypothesis, candidates)
	if err != nil {
		return Fix{}, fmt.Errorf("build fix prompt: %w", err)
	}

	text, err := c.analyze(ctx, prompt, "")
	if err != nil {
		return Fix{}, fmt.Errorf("claude: %w", err)
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

func cleanJSON(text string) string {
	text = strings.TrimSpace(text)
	if start := strings.Index(text, "{"); start >= 0 {
		text = text[start:]
	}
	if end := strings.LastIndex(text, "}"); end >= 0 {
		text = text[:end+1]
	}
	return text
}
