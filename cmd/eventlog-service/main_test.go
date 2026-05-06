package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
	"github.com/example/onec-eventlog-mvp/internal/config"
	"github.com/example/onec-eventlog-mvp/internal/reader"
	"github.com/example/onec-eventlog-mvp/internal/status"
)

func TestFileModeDrainWaitsUntilNoPendingEvents(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := buffer.Open(filepath.Join(tmpDir, "buffer.db"))
	if err != nil {
		t.Fatalf("open buffer: %v", err)
	}
	defer store.Close()

	jsonlPath := filepath.Join(tmpDir, "events.jsonl")
	if err := os.WriteFile(jsonlPath, []byte("{\"Date\":\"2026-05-01T10:00:00Z\"}\n{\"Date\":\"2026-05-01T10:01:00Z\"}\n"), 0o644); err != nil {
		t.Fatalf("write jsonl: %v", err)
	}

	source := config.SourceConfig{
		SourceID:     "source-1",
		SourceNodeID: "node-1",
		InfoBaseID:   "ib-1",
		Mode:         "file",
		FilePath:     jsonlPath,
	}
	state := status.New("test-service", source.SourceID, source.InfoBaseID)
	fileReader := reader.New(source, store, state)

	if err := fileReader.Run(context.Background()); err != nil {
		t.Fatalf("run file reader: %v", err)
	}
	beforeDrain, err := store.PendingCount(context.Background())
	if err != nil {
		t.Fatalf("pending count before drain: %v", err)
	}
	if beforeDrain != 2 {
		t.Fatalf("pending count before drain = %d, want 2", beforeDrain)
	}

	drainStarted := make(chan struct{})
	go func() {
		<-drainStarted
		time.Sleep(20 * time.Millisecond)

		pending, err := store.FetchPending(context.Background(), 10)
		if err != nil {
			t.Errorf("fetch pending: %v", err)
			return
		}
		if len(pending) != 2 {
			t.Errorf("expected 2 pending events, got %d", len(pending))
			return
		}
		localIDs := make([]int64, 0, len(pending))
		for _, msg := range pending {
			localIDs = append(localIDs, msg.LocalID)
		}
		if err := store.MarkDone(context.Background(), localIDs); err != nil {
			t.Errorf("mark done: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	close(drainStarted)
	if err := waitForBufferDrain(ctx, store, 5*time.Millisecond); err != nil {
		t.Fatalf("wait for drain: %v", err)
	}

	count, err := store.PendingCount(context.Background())
	if err != nil {
		t.Fatalf("pending count: %v", err)
	}
	if count != 0 {
		t.Fatalf("pending count = %d, want 0", count)
	}
}
