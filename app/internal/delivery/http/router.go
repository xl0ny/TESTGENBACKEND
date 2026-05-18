package http

import (
	"github.com/gin-gonic/gin"
)

func NewRouter(h *Handlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), corsMiddleware())

	r.POST("/api/session", h.PostSession)
	r.DELETE("/api/session", h.DeleteSession)
	r.POST("/api/generate", h.PostGenerate)
	r.POST("/api/result", h.PostResult)
	r.POST("/v1/messages/send", h.ProxySend)
	r.POST("/v1/messages/receive", h.ProxyReceive)
	r.GET("/health", h.Health)

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
