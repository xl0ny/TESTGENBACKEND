package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/testgen/app/internal/domain"
	"github.com/testgen/app/internal/usecase"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	hub        domain.ClientHub
	generation *usecase.GenerationUseCase
	chat       *usecase.ChatUseCase
}

func NewHandler(hub domain.ClientHub, generation *usecase.GenerationUseCase, chat *usecase.ChatUseCase) *Handler {
	return &Handler{hub: hub, generation: generation, chat: chat}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := NewConn(raw)
	var username string

	defer func() {
		if username != "" {
			h.hub.Remove(username, conn)
		}
		_ = raw.Close()
	}()

	for {
		_, data, err := raw.ReadMessage()
		if err != nil {
			return
		}

		var msg map[string]any
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = conn.SendJSON(map[string]any{"type": "error", "message": "invalid json"})
			continue
		}

		msgType, _ := msg["type"].(string)

		if msgType == "auth" || msgType == "login" {
			username = strings.TrimSpace(firstString(msg, "username", "sender"))
			if username == "" {
				_ = conn.SendJSON(map[string]any{"type": "error", "message": "username required"})
				continue
			}
			h.hub.Add(username, conn)
			_ = conn.SendJSON(map[string]any{"type": "auth_ok", "sender": username})
			continue
		}

		if msgType == "logout" {
			if username != "" {
				h.hub.Remove(username, conn)
				username = ""
			}
			_ = raw.Close()
			return
		}

		if username == "" {
			_ = conn.SendJSON(map[string]any{"type": "error", "message": "authenticate first"})
			continue
		}

		switch msgType {
		case "generate":
			schema := msg["json_schema"]
			sampleCount := intFromAny(msg["sample_count"])
			if schema == nil || sampleCount < 1 {
				_ = conn.SendJSON(map[string]any{"type": "error", "message": "json_schema and sample_count required"})
				continue
			}
			constraints, _ := msg["constraints"].(string)
			sentAt, _ := msg["sent_at"].(string)
			err := h.generation.StartWSGeneration(r.Context(), username, domain.GenerateRequest{
				SentAt:      sentAt,
				JSONSchema:  schema,
				SampleCount: sampleCount,
				Constraints: constraints,
			})
			if err != nil {
				_ = conn.SendJSON(map[string]any{"type": "error", "message": err.Error()})
			}
		case "send":
			payload, _ := msg["payload"].(string)
			sentAt, _ := msg["sent_at"].(string)
			h.chat.SendLocal(username, payload, sentAt)
		default:
			_ = conn.SendJSON(map[string]any{"type": "error", "message": "unknown message type"})
		}
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

// Register mounts /ws on the given mux (stdlib) for use alongside Gin.
func Register(mux *http.ServeMux, handler *Handler) {
	mux.Handle("/ws", handler)
}

func ListenAndServe(addr string, ginHandler http.Handler, wsHandler *Handler) error {
	mux := http.NewServeMux()
	mux.Handle("/", ginHandler)
	Register(mux, wsHandler)
	log.Printf("app backend listening on %s (ws /ws, rest /api)", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}
