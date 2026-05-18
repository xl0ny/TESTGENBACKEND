package main

import (
	"log"

	"github.com/testgen/app/internal/delivery/http"
	"github.com/testgen/app/internal/delivery/ws"
	"github.com/testgen/app/internal/infrastructure/config"
	"github.com/testgen/app/internal/infrastructure/hub"
	transportclient "github.com/testgen/app/internal/infrastructure/transport"
	"github.com/testgen/app/internal/usecase"
)

func main() {
	cfg := config.Load()

	clientHub := hub.NewMemoryHub()
	transport := transportclient.NewClient(cfg.TransportBaseURL)

	poll := usecase.ReceivePollSettings{
		WaitMS:         cfg.ReceiveWaitMS,
		MaxAttempts:    cfg.ReceiveMaxAttempts,
		PollIntervalMS: cfg.ReceivePollIntervalMS,
	}

	sessionUC := usecase.NewSessionUseCase()
	generationUC := usecase.NewGenerationUseCase(transport, clientHub, poll)
	proxyUC := usecase.NewProxyUseCase(transport)
	chatUC := usecase.NewChatUseCase(clientHub)

	handlers := http.NewHandlers(sessionUC, generationUC, proxyUC, transport)
	router := http.NewRouter(handlers)
	wsHandler := ws.NewHandler(clientHub, generationUC, chatUC)

	if err := ws.ListenAndServe(cfg.HTTPAddr, router, wsHandler); err != nil {
		log.Fatal(err)
	}
}
