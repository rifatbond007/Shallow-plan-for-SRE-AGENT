package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port            int             `env:"SRE_AGENT_PORT" envDefault:"8080"`
	LogLevel        string          `env:"SRE_AGENT_LOG_LEVEL" envDefault:"info"`
	LLMProvider     string          `env:"SRE_AGENT_LLM_PROVIDER" envDefault:"claude"`
	Anthropic       AnthropicConfig `envPrefix:"SRE_AGENT_ANTHROPIC_"`
	Gemini          GeminiConfig    `envPrefix:"SRE_AGENT_GEMINI_"`
	CodebasePath    string          `env:"SRE_AGENT_CODEBASE_PATH" envDefault:"/codebase"`
	CodebaseCache   string          `env:"SRE_AGENT_CODEBASE_CACHE_DIR" envDefault:"/tmp/sre-agent/cache"`
	MaxLogBytes     int             `env:"SRE_AGENT_MAX_LOG_BYTES" envDefault:"5000000"`
	IncidentWindow  time.Duration   `env:"SRE_AGENT_INCIDENT_WINDOW" envDefault:"5m"`
	IncidentMinSize int             `env:"SRE_AGENT_INCIDENT_MIN_SIZE" envDefault:"3"`
	HypothesisTopK  int             `env:"SRE_AGENT_HYPOTHESIS_TOP_K" envDefault:"3"`
	CacheTTL        time.Duration   `env:"SRE_AGENT_CACHE_TTL" envDefault:"1h"`
	CacheMaxEntries int             `env:"SRE_AGENT_CACHE_MAX_ENTRIES" envDefault:"512"`
	RateLimitRPS    float64         `env:"SRE_AGENT_RATE_LIMIT_RPS" envDefault:"5"`
	RateLimitBurst  int             `env:"SRE_AGENT_RATE_LIMIT_BURST" envDefault:"10"`
	APIKey          string          `env:"SRE_AGENT_API_KEY" envDefault:""`
}

type AnthropicConfig struct {
	APIKey    string        `env:"API_KEY" envDefault:""`
	Model     string        `env:"MODEL" envDefault:"claude-sonnet-4-20250514"`
	MaxTokens int           `env:"MAX_TOKENS" envDefault:"2048"`
	Timeout   time.Duration `env:"TIMEOUT" envDefault:"30s"`
}

type GeminiConfig struct {
	APIKey    string        `env:"API_KEY" envDefault:""`
	Model     string        `env:"MODEL" envDefault:"gemini-2.0-flash"`
	MaxTokens int           `env:"MAX_TOKENS" envDefault:"2048"`
	Timeout   time.Duration `env:"TIMEOUT" envDefault:"30s"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}
