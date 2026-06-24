package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func requestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		)
	}
}

func authMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() == "/api/v1/healthz" || c.FullPath() == "/api/v1/readyz" || c.FullPath() == "/metrics" {
			c.Next()
			return
		}
		key := c.GetHeader("X-API-Key")
		if key == "" {
			key = c.Query("api_key")
		}
		if key != apiKey {
			respondError(c, 401, "INVALID_REQUEST", "missing or invalid API key")
			c.Abort()
			return
		}
		c.Next()
	}
}

func rateLimiter(rps float64, burst int) gin.HandlerFunc {
	tokens := make(chan struct{}, burst)
	go func() {
		ticker := time.NewTicker(time.Duration(1e9 / rps))
		defer ticker.Stop()
		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
			}
		}
	}()

	return func(c *gin.Context) {
		select {
		case <-tokens:
			c.Next()
		default:
			respondError(c, 429, "INVALID_REQUEST", "rate limit exceeded")
			c.Abort()
		}
	}
}
