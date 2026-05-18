package usecase

import (
	"context"
	"encoding/json"

	"github.com/testgen/app/internal/domain"
)

type ProxyUseCase struct {
	transport domain.TransportClient
}

func NewProxyUseCase(transport domain.TransportClient) *ProxyUseCase {
	return &ProxyUseCase{transport: transport}
}

func (uc *ProxyUseCase) Send(ctx context.Context, body map[string]any) (domain.SendAccepted, error) {
	if body == nil {
		body = map[string]any{}
	}
	delete(body, "password")
	return uc.transport.Send(ctx, body)
}

func (uc *ProxyUseCase) Receive(ctx context.Context, body map[string]any) (domain.ReceiveResult, error) {
	if body == nil {
		body = map[string]any{}
	}
	data, _ := json.Marshal(body)
	var query domain.ReceiveQuery
	_ = json.Unmarshal(data, &query)
	return uc.transport.Receive(ctx, query)
}
