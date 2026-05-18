package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/testgen/app/internal/domain"
	"github.com/testgen/app/internal/usecase"
)

type Handlers struct {
	session    *usecase.SessionUseCase
	generation *usecase.GenerationUseCase
	proxy      *usecase.ProxyUseCase
	transport  domain.TransportClient
}

func NewHandlers(
	session *usecase.SessionUseCase,
	generation *usecase.GenerationUseCase,
	proxy *usecase.ProxyUseCase,
	transport domain.TransportClient,
) *Handlers {
	return &Handlers{
		session:    session,
		generation: generation,
		proxy:      proxy,
		transport:  transport,
	}
}

func (h *Handlers) PostSession(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation", "message": "email required"})
		return
	}
	sender, err := h.session.CreateSession(body.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation", "message": "email required"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sender": sender})
}

func (h *Handlers) DeleteSession(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func (h *Handlers) PostGenerate(c *gin.Context) {
	var body struct {
		Sender      string         `json:"sender"`
		JSONSchema  map[string]any `json:"json_schema"`
		SampleCount int            `json:"sample_count"`
		Constraints string         `json:"constraints"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation", "message": "sender, json_schema, sample_count required"})
		return
	}
	messageID, err := h.generation.GenerateREST(c.Request.Context(), domain.GenerateRequest{
		Sender:      body.Sender,
		JSONSchema:  body.JSONSchema,
		SampleCount: body.SampleCount,
		Constraints: body.Constraints,
	})
	if errors.Is(err, usecase.ErrGenerateValidation) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "validation", "message": "sender, json_schema, sample_count required"})
		return
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "transport_unavailable", "message": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message_id": messageID})
}

func (h *Handlers) PostResult(c *gin.Context) {
	var body struct {
		Sender string `json:"sender"`
		WaitMs int    `json:"wait_ms"`
	}
	_ = c.ShouldBindJSON(&body)
	result, err := h.generation.ReceiveResult(c.Request.Context(), body.Sender, body.WaitMs)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "transport_unavailable", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handlers) ProxySend(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		body = map[string]any{}
	}
	accepted, err := h.proxy.Send(c.Request.Context(), body)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, accepted)
}

func (h *Handlers) ProxyReceive(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		body = map[string]any{}
	}
	result, err := h.proxy.Receive(c.Request.Context(), body)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handlers) Health(c *gin.Context) {
	transportOk, _ := h.transport.Health(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"status": "ok", "transport": transportOk})
}
