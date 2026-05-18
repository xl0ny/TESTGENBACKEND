package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/testgen/app/internal/domain"
)

var (
	ErrGenerateValidation = errors.New("sender, json_schema, sample_count required")
	ErrWSGenerateValidation = errors.New("json_schema and sample_count required")
)

type ReceivePollSettings struct {
	WaitMS         int
	MaxAttempts    int
	PollIntervalMS int
}

type GenerationUseCase struct {
	transport domain.TransportClient
	hub       domain.ClientHub
	poll      ReceivePollSettings
}

func NewGenerationUseCase(transport domain.TransportClient, hub domain.ClientHub, poll ReceivePollSettings) *GenerationUseCase {
	return &GenerationUseCase{transport: transport, hub: hub, poll: poll}
}

func (uc *GenerationUseCase) GenerateREST(ctx context.Context, req domain.GenerateRequest) (string, error) {
	if req.Sender == "" || req.JSONSchema == nil || req.SampleCount < 1 {
		return "", ErrGenerateValidation
	}
	sentAt := req.SentAt
	if sentAt == "" {
		sentAt = time.Now().UTC().Format(time.RFC3339)
	}
	accepted, err := uc.transport.Send(ctx, map[string]any{
		"sender":       req.Sender,
		"sent_at":      sentAt,
		"json_schema":  req.JSONSchema,
		"sample_count": req.SampleCount,
		"constraints":  req.Constraints,
	})
	if err != nil {
		return "", err
	}
	return accepted.MessageID, nil
}

func (uc *GenerationUseCase) ReceiveResult(ctx context.Context, sender string, waitMs int) (domain.ReceiveResult, error) {
	if waitMs <= 0 {
		waitMs = 10000
	}
	return uc.transport.Receive(ctx, domain.ReceiveQuery{
		Sender: sender,
		WaitMs: waitMs,
	})
}

func (uc *GenerationUseCase) StartWSGeneration(ctx context.Context, username string, req domain.GenerateRequest) error {
	if req.JSONSchema == nil || req.SampleCount < 1 {
		return ErrWSGenerateValidation
	}
	sentAt := req.SentAt
	if sentAt == "" {
		sentAt = time.Now().UTC().Format(time.RFC3339)
	}

	accepted, err := uc.transport.Send(ctx, map[string]any{
		"sender":       username,
		"sent_at":      sentAt,
		"json_schema":  req.JSONSchema,
		"sample_count": req.SampleCount,
		"constraints":  req.Constraints,
	})
	if err != nil {
		return err
	}

	uc.hub.PushToUser(username, map[string]any{
		"type":       "status",
		"message_id": accepted.MessageID,
		"status":     "processing",
	})

	go uc.finishWSGeneration(username, accepted.MessageID, sentAt)
	return nil
}

func (uc *GenerationUseCase) finishWSGeneration(username, messageID, sentAt string) {
	ctx := context.Background()
	result, err := uc.streamGenerationResults(ctx, username, messageID)
	if err != nil {
		uc.deliverGenerationResult(username, domain.ReceiveResult{
			Sender:  username,
			SentAt:  sentAt,
			Error:   true,
			Payload: "",
		}, messageID, err.Error())
		return
	}
	if result.Sender == "" {
		uc.deliverGenerationResult(username, domain.ReceiveResult{
			Sender:  username,
			SentAt:  sentAt,
			Error:   true,
			Payload: "",
		}, messageID, "generation timeout")
	}
}

func (uc *GenerationUseCase) deliverGenerationResult(username string, result domain.ReceiveResult, messageID, errMsg string) {
	sender := result.Sender
	if sender == "" {
		sender = username
	}
	msg := map[string]any{
		"type":       "receive",
		"sender":     sender,
		"sent_at":    result.SentAt,
		"error":      result.Error,
		"payload":    result.Payload,
		"message_id": messageID,
	}
	if errMsg != "" {
		msg["message"] = errMsg
	}
	uc.hub.PushToUser(username, msg)
	uc.hub.BroadcastExcept(sender, msg)
}

func (uc *GenerationUseCase) streamGenerationResults(ctx context.Context, sender, messageID string) (domain.ReceiveResult, error) {
	interval := time.Duration(uc.poll.PollIntervalMS) * time.Millisecond
	emptyAfterData := 0
	var last domain.ReceiveResult

	for i := 0; i < uc.poll.MaxAttempts; i++ {
		result, err := uc.transport.Receive(ctx, domain.ReceiveQuery{
			Sender: sender,
			WaitMs: uc.poll.WaitMS,
		})
		if err != nil {
			return domain.ReceiveResult{}, err
		}
		if result.Sender != "" {
			last = result
			emptyAfterData = 0
			uc.deliverGenerationResult(sender, result, messageID, "")
			if result.Error {
				return result, nil
			}
			select {
			case <-ctx.Done():
				return domain.ReceiveResult{}, ctx.Err()
			case <-time.After(interval):
			}
			continue
		}
		if last.Sender != "" {
			emptyAfterData++
			if emptyAfterData >= 2 {
				return last, nil
			}
		}
		select {
		case <-ctx.Done():
			return domain.ReceiveResult{}, ctx.Err()
		case <-time.After(interval):
		}
	}
	if last.Sender != "" {
		return last, nil
	}
	return domain.ReceiveResult{}, nil
}

func (uc *GenerationUseCase) pollGenerationResult(ctx context.Context, sender string) (domain.ReceiveResult, error) {
	result, err := uc.streamGenerationResults(ctx, sender, "")
	if err != nil {
		return domain.ReceiveResult{}, err
	}
	if result.Sender == "" {
		return domain.ReceiveResult{
			Sender:  sender,
			SentAt:  time.Now().UTC().Format(time.RFC3339),
			Error:   true,
			Payload: "",
		}, nil
	}
	return result, nil
}
