package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondError(c *gin.Context, status int, code, msg string) {
	c.JSON(status, gin.H{"error": APIError{Code: code, Message: msg}})
}

var errorStatus = map[string]int{
	"INVALID_REQUEST":    http.StatusBadRequest,
	"LOGS_TOO_LARGE":    http.StatusRequestEntityTooLarge,
	"LLM_UPSTREAM":      http.StatusBadGateway,
	"TIMEOUT":           http.StatusGatewayTimeout,
	"NOT_FOUND":         http.StatusNotFound,
	"INTERNAL":          http.StatusInternalServerError,
}
