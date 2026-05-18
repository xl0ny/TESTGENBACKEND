package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/testgen/transport/internal/agent"
	"github.com/testgen/transport/internal/api"
	"github.com/testgen/transport/internal/assembly"
	"github.com/testgen/transport/internal/config"
	"github.com/testgen/transport/internal/kafka"
	"github.com/testgen/transport/internal/store"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cfg := config.Load()
	st := store.New()
	kc := kafka.NewClient(cfg, st)
	ag := agent.NewClient(cfg)
	srv := api.NewServer(cfg, st, kc, ag)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go kc.ConsumeLoop(ctx)

	stopAsm := make(chan struct{})
	go assembly.NewRunner(cfg, st).Start(stopAsm)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("transport listening on %s (loss=%.0f%%, assembly=%s)", cfg.HTTPAddr, cfg.SegmentLossRate*100, cfg.AssemblyInterval)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	close(stopAsm)
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	_ = kc.Close()
}
