package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
	chwriter "github.com/example/onec-eventlog-mvp/internal/clickhouse"
	"github.com/example/onec-eventlog-mvp/internal/config"
	"github.com/example/onec-eventlog-mvp/internal/reader"
	"github.com/example/onec-eventlog-mvp/internal/status"
)

func main() {
	configPath := flag.String("config", "config.example.yaml", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	state := status.New(cfg.Service.Name, cfg.Source.SourceID, cfg.Source.InfoBaseID)

	bufferStore, err := buffer.Open(cfg.Buffer.Path)
	if err != nil {
		log.Fatalf("open buffer: %v", err)
	}
	defer bufferStore.Close()

	writer, err := chwriter.Open(ctx, cfg.ClickHouse)
	if err != nil {
		log.Fatalf("open clickhouse writer: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", state.Handler())
	mux.HandleFunc("/api/events", writer.EventsHandler())
	mux.HandleFunc("/api/events/", writer.EventDetailHandler())

	httpServer := &http.Server{
		Addr:    cfg.Service.HTTPAddr,
		Handler: mux,
	}
	go func() {
		log.Printf("status http server listening on %s", cfg.Service.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			state.SetError(err)
		}
	}()

	jrReader := reader.New(cfg.Source, bufferStore, state)
	worker := chwriter.NewWorker(bufferStore, writer, cfg.Buffer.BatchSize, state)

	go func() {
		if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			state.SetError(err)
		}
	}()

	log.Printf("starting ЖР reader for source=%s infobase=%s", cfg.Source.SourceID, cfg.Source.InfoBaseID)
	if err := jrReader.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		state.SetError(err)
		log.Printf("reader stopped with error: %v", err)
	}
	if cfg.Source.Mode == "file" && ctx.Err() == nil {
		if err := waitForBufferDrain(ctx, bufferStore, time.Second); err != nil && !errors.Is(err, context.Canceled) {
			state.SetError(err)
			log.Printf("wait for buffer drain stopped with error: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	state.SetStatus("stopped")
}

func waitForBufferDrain(ctx context.Context, bufferStore *buffer.Store, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		pending, err := bufferStore.PendingCount(ctx)
		if err != nil {
			return err
		}
		if pending == 0 {
			log.Print("file reader drained sqlite buffer")
			return nil
		}
		log.Printf("waiting for sqlite buffer drain: pending=%d", pending)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
