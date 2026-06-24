package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/rifatbond007/sre-ai-agent/internal/config"
	"github.com/rifatbond007/sre-ai-agent/pkg/logger"
	"github.com/rifatbond007/sre-ai-agent/pkg/metrics"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	zapLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := strconv.Itoa(c.Writer.Status())
		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(time.Since(start).Seconds())
	})

	r.GET("/api/v1/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/api/v1/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if cfg.APIKey != "" {
		zapLog.Info("API key auth enabled for /api/v1/analyze")
	} else {
		zapLog.Warn("no SRE_AGENT_API_KEY set — /api/v1/analyze is open")
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
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
