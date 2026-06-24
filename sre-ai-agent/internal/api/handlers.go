package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rifatbond007/sre-ai-agent/internal/analysis"
	"github.com/rifatbond007/sre-ai-agent/internal/storage"
	"github.com/rifatbond007/sre-ai-agent/pkg/metrics"
	"go.uber.org/zap"
)

type handlers struct {
	engine      analysis.Engine
	store       *storage.Store
	maxLogBytes int
	log         *zap.Logger
}

type analyzeRequest struct {
	Logs         string `json:"logs"`
	CodebasePath string `json:"codebase_path"`
	TopK         int    `json:"top_k"`
}

func (h *handlers) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *handlers) readyz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *handlers) analyze(c *gin.Context) {
	metrics.HTTPRequestsTotal.WithLabelValues("POST", "/analyze", "200")

	body := c.Request.Body
	body = http.MaxBytesReader(c.Writer, body, int64(h.maxLogBytes))

	var req analyzeRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			respondError(c, http.StatusRequestEntityTooLarge, "LOGS_TOO_LARGE", "request body exceeds maximum size")
			return
		}
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	if req.Logs == "" {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "logs field is required")
		return
	}
	if req.CodebasePath == "" {
		req.CodebasePath = "/codebase"
	}
	if req.TopK <= 0 {
		req.TopK = 3
	}

	resultID := uuid.New().String()

	ar, err := h.engine.Analyze(c.Request.Context(), analysis.AnalyzeRequest{
		Logs:         req.Logs,
		CodebasePath: req.CodebasePath,
		TopK:         req.TopK,
	})
	if err != nil {
		h.log.Error("analysis failed", zap.Error(err))
		code := "INTERNAL"
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "claude") || strings.Contains(err.Error(), "Claude") {
			code = "CLAUDE_UPSTREAM"
			status = http.StatusBadGateway
		}
		respondError(c, status, code, err.Error())
		return
	}

	ar.ID = resultID
	h.store.Set(resultID, ar)

	c.JSON(http.StatusOK, ar)
}

func (h *handlers) getHypothesis(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "id is required")
		return
	}

	result, ok := h.store.Get(id)
	if !ok {
		respondError(c, http.StatusNotFound, "NOT_FOUND", "result not found or expired")
		return
	}

	metrics.CacheHitsTotal.Inc()
	c.JSON(http.StatusOK, result)
}
