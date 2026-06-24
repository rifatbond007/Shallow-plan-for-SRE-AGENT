package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/rifatbond007/sre-ai-agent/internal/api"
	"github.com/rifatbond007/sre-ai-agent/internal/analysis"
	"github.com/rifatbond007/sre-ai-agent/internal/config"
	"github.com/rifatbond007/sre-ai-agent/internal/storage"
	"github.com/rifatbond007/sre-ai-agent/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	zapLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}

	var llm analysis.LLMClient
	switch cfg.LLMProvider {
	case "gemini":
		llm = analysis.NewGeminiClient(
			cfg.Gemini.APIKey,
			cfg.Gemini.Model,
			cfg.Gemini.Timeout,
		)
		zapLog.Info("using Gemini LLM provider", zap.String("model", cfg.Gemini.Model))
	default:
		llm = analysis.NewClaudeClient(
			cfg.Anthropic.APIKey,
			cfg.Anthropic.Model,
			cfg.Anthropic.Timeout,
		)
		zapLog.Info("using Claude LLM provider", zap.String("model", cfg.Anthropic.Model))
	}

	engine := analysis.NewEngine(llm, cfg.MaxLogBytes)

	store := storage.NewStore(cfg.CacheMaxEntries, cfg.CacheTTL)

	router := api.NewRouter(engine, store, zapLog, cfg.APIKey, cfg.MaxLogBytes)

	if cfg.APIKey != "" {
		zapLog.Info("API key auth enabled for /api/v1/analyze")
	} else {
		zapLog.Warn("no SRE_AGENT_API_KEY set — /api/v1/analyze is open")
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		zapLog.Info("starting server", zap.Int("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLog.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLog.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLog.Fatal("forced shutdown", zap.Error(err))
	}
	zapLog.Info("stopped")
}
