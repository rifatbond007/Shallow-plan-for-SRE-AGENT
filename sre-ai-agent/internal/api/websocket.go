package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	gorilla "github.com/gorilla/websocket"
	"github.com/google/uuid"
	"github.com/rifatbond007/sre-ai-agent/internal/analysis"
	"go.uber.org/zap"
)

var upgrader = gorilla.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *handlers) analyzeStreamWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	_, msg, err := conn.ReadMessage()
	if err != nil {
		writeWSError(conn, "INVALID_REQUEST", fmt.Sprintf("read: %v", err))
		return
	}

	if len(msg) > h.maxLogBytes {
		writeWSError(conn, "LOGS_TOO_LARGE", fmt.Sprintf("message exceeds max size of %d bytes", h.maxLogBytes))
		return
	}

	var req analyzeRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		writeWSError(conn, "INVALID_REQUEST", "invalid JSON")
		return
	}

	if req.Logs == "" {
		writeWSError(conn, "INVALID_REQUEST", "logs field is required")
		return
	}
	if req.CodebasePath == "" {
		req.CodebasePath = "/codebase"
	}
	if req.TopK <= 0 {
		req.TopK = 3
	}

	resultID := uuid.New().String()

	var mu sync.Mutex
	sink := func(evt analysis.ProgressEvent) error {
		mu.Lock()
		defer mu.Unlock()
		return conn.WriteJSON(evt)
	}

	ar, err := h.engine.AnalyzeStream(c.Request.Context(), analysis.AnalyzeRequest{
		Logs:         req.Logs,
		CodebasePath: req.CodebasePath,
		TopK:         req.TopK,
	}, sink)
	if err != nil {
		code := "INTERNAL"
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "claude") || strings.Contains(errStr, "gemini") || strings.Contains(errStr, "anthropic") {
			code = "LLM_UPSTREAM"
		}
		writeWSError(conn, code, err.Error())
		return
	}

	ar.ID = resultID
	h.store.Set(resultID, ar)

	mu.Lock()
	conn.WriteJSON(analysis.ProgressEvent{Type: "result", Data: ar})
	mu.Unlock()
}

func writeWSError(conn *gorilla.Conn, code, msg string) {
	evt := analysis.ProgressEvent{
		Type:  "error",
		Error: msg,
	}
	if err := conn.WriteJSON(evt); err != nil {
		log.Printf("WebSocket write error: %v", err)
	}
}
