package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rifatbond007/sre-ai-agent/internal/analysis"
	"github.com/rifatbond007/sre-ai-agent/internal/storage"
	"go.uber.org/zap"
)

func NewRouter(engine analysis.Engine, store *storage.Store, log *zap.Logger, apiKey string, maxLogBytes int) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(log))

	h := &handlers{
		engine:      engine,
		store:       store,
		maxLogBytes: maxLogBytes,
		log:         log,
	}

	if apiKey != "" {
		r.Use(authMiddleware(apiKey))
	}

	r.GET("/api/v1/healthz", h.healthz)
	r.GET("/api/v1/readyz", h.readyz)
	r.POST("/api/v1/analyze", h.analyze)
	r.GET("/api/v1/hypotheses/:id", h.getHypothesis)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return r
}
