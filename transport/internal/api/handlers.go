package api

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/testgen/transport/internal/agent"
	"github.com/testgen/transport/internal/config"
	"github.com/testgen/transport/internal/kafka"
	"github.com/testgen/transport/internal/models"
	"github.com/testgen/transport/internal/store"
)

type Server struct {
	cfg    config.Config
	store  *store.Store
	kafka  *kafka.Client
	agent  *agent.Client
}

func NewServer(cfg config.Config, st *store.Store, k *kafka.Client, ag *agent.Client) *Server {
	return &Server{cfg: cfg, store: st, kafka: k, agent: ag}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/segments", s.postSegment)
	mux.HandleFunc("POST /v1/generation/failed", s.postGenerationFailed)
	mux.HandleFunc("POST /v1/messages/send", s.postSend)
	mux.HandleFunc("POST /v1/messages/receive", s.postReceive)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return withCORS(mux)
}

func (s *Server) postSegment(w http.ResponseWriter, r *http.Request) {
	var seg models.Segment
	if err := json.NewDecoder(r.Body).Decode(&seg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if seg.MessageID == "" || seg.TotalSegments < 1 || seg.SegmentIndex < 0 || seg.PayloadChunk == "" {
		writeError(w, http.StatusBadRequest, "validation", "missing required segment fields")
		return
	}

	if rand.Float64() < s.cfg.SegmentLossRate {
		log.Printf("segment lost message=%s index=%d (R=%.0f%%)", seg.MessageID, seg.SegmentIndex, s.cfg.SegmentLossRate*100)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	meta, _ := s.store.Meta(seg.MessageID)
	if meta.Sender == "" {
		meta = store.MessageMeta{SentAt: seg.SentAt}
	}
	s.store.EnsureAssembly(seg.MessageID, seg.TotalSegments, meta)

	if err := s.kafka.ProduceSegment(r.Context(), seg); err != nil {
		log.Printf("kafka produce: %v", err)
		writeError(w, http.StatusInternalServerError, "kafka", "failed to enqueue segment")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) postGenerationFailed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MessageID string `json:"message_id"`
		Sender    string `json:"sender"`
		SentAt    string `json:"sent_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if body.MessageID == "" {
		writeError(w, http.StatusBadRequest, "validation", "message_id required")
		return
	}
	meta, ok := s.store.Meta(body.MessageID)
	if body.Sender == "" && ok {
		body.Sender = meta.Sender
	}
	if body.SentAt == "" && ok {
		body.SentAt = meta.SentAt
	}
	s.store.RemoveAssembly(body.MessageID)
	s.store.EnqueueOutbound(models.OutboundMessage{
		MessageID: body.MessageID,
		Sender:    body.Sender,
		SentAt:    body.SentAt,
		Error:     true,
		Payload:   "",
		ReadyAt:   time.Now(),
	})
	log.Printf("generation failed for message %s", body.MessageID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) postSend(w http.ResponseWriter, r *http.Request) {
	var payload models.SendPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if payload.Sender == "" || payload.SentAt == "" || payload.JSONSchema == nil || payload.SampleCount < 1 {
		writeError(w, http.StatusBadRequest, "validation", "sender, sent_at, json_schema and sample_count are required")
		return
	}

	messageID := uuid.NewString()
	s.store.PutMeta(messageID, payload.Sender, payload.SentAt)

	accepted := models.SendAccepted{MessageID: messageID}
	writeJSON(w, http.StatusAccepted, accepted)

	go func() {
		task := models.GenerationTaskRequest{
			MessageID:   messageID,
			Sender:      payload.Sender,
			SentAt:      payload.SentAt,
			JSONSchema:  payload.JSONSchema,
			SampleCount: payload.SampleCount,
			Constraints: payload.Constraints,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := s.agent.CreateGenerationTask(ctx, task); err != nil {
			log.Printf("agent task failed for %s: %v", messageID, err)
			s.store.EnqueueOutbound(models.OutboundMessage{
				MessageID: messageID,
				Sender:    payload.Sender,
				SentAt:    payload.SentAt,
				Error:     true,
				Payload:   "",
				ReadyAt:   time.Now(),
			})
		}
	}()
}

func (s *Server) postReceive(w http.ResponseWriter, r *http.Request) {
	var query models.ReceiveQuery
	if r.Body != nil && r.ContentLength != 0 {
		_ = json.NewDecoder(r.Body).Decode(&query)
	}

	wait := time.Duration(query.WaitMs) * time.Millisecond
	if wait <= 0 {
		wait = 100 * time.Millisecond
	}
	if wait > 30*time.Second {
		wait = 30 * time.Second
	}

	deadline := time.Now().Add(wait)
	for {
		if msg, ok := s.store.DequeueOutbound(query.Sender); ok {
			writeJSON(w, http.StatusOK, models.ReceivePayload{
				Sender:  msg.Sender,
				SentAt:  msg.SentAt,
				Error:   msg.Error,
				Payload: msg.Payload,
			})
			return
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		s.store.WaitOutbound(remaining)
	}

	writeJSON(w, http.StatusOK, models.ReceivePayload{
		Sender:  "",
		SentAt:  time.Now().UTC().Format(time.RFC3339),
		Error:   false,
		Payload: "",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.ErrorBody{Code: code, Message: message})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
