package usecase

import (
	"time"

	"github.com/testgen/app/internal/domain"
)

type ChatUseCase struct {
	hub domain.ClientHub
}

func NewChatUseCase(hub domain.ClientHub) *ChatUseCase {
	return &ChatUseCase{hub: hub}
}

func (uc *ChatUseCase) SendLocal(username, payload, sentAt string) {
	if sentAt == "" {
		sentAt = time.Now().UTC().Format(time.RFC3339)
	}
	msg := map[string]any{
		"type":    "receive",
		"sender":  username,
		"sent_at": sentAt,
		"error":   false,
		"payload": payload,
	}
	uc.hub.PushToUser(username, msg)
	uc.hub.BroadcastExcept(username, msg)
}
