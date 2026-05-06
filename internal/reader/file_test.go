package reader_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
	"github.com/example/onec-eventlog-mvp/internal/config"
	"github.com/example/onec-eventlog-mvp/internal/reader"
	"github.com/example/onec-eventlog-mvp/internal/status"
)

func TestFileReader_ReadJSONL(t *testing.T) {
	// Setup: create temporary buffer database and test JSONL file
	tmpDir := t.TempDir()
	bufferPath := filepath.Join(tmpDir, "test.db")
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")

	// Create a simple JSONL file with 3 events
	jsonlContent := `{"Date":"2026-05-01T10:00:00","User":"admin","Computer":"PC01"}
{"Date":"2026-05-01T10:01:00","User":"user1","Computer":"PC02"}
{"Date":"2026-05-01T10:02:00","User":"user2","Computer":"PC03"}
`
	if err := os.WriteFile(jsonlPath, []byte(jsonlContent), 0o644); err != nil {
		t.Fatalf("write test jsonl file: %v", err)
	}

	// Open buffer store
	store, err := buffer.Open(bufferPath)
	if err != nil {
		t.Fatalf("open buffer: %v", err)
	}
	defer store.Close()

	// Create config
	cfg := config.SourceConfig{
		SourceID:     "test-source",
		InfoBaseID:   "test-infobase",
		SourceNodeID: "test-node",
		FilePath:     jsonlPath,
		Mode:         "file",
	}

	// Create status store
	statusStore := status.New("test-service", cfg.SourceID, cfg.InfoBaseID)

	// Create reader and run
	r := reader.New(cfg, store, statusStore)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.Run(ctx); err != nil {
		t.Fatalf("reader run: %v", err)
	}

	// Verify that 3 events were read and buffered
	snapshot := statusStore.Snapshot()
	if snapshot.ReadEvents != 3 {
		t.Errorf("expected 3 read events, got %d", snapshot.ReadEvents)
	}
	if snapshot.BufferedEvents != 3 {
		t.Errorf("expected 3 buffered events, got %d", snapshot.BufferedEvents)
	}

	// Verify that events are in the buffer
	pending, err := store.FetchPending(ctx, 10)
	if err != nil {
		t.Fatalf("fetch pending: %v", err)
	}
	if len(pending) != 3 {
		t.Errorf("expected 3 pending events in buffer, got %d", len(pending))
	}

	// Verify the content of the first event
	if len(pending) > 0 {
		firstEvent := pending[0]
		if firstEvent.SourceID != "test-source" {
			t.Errorf("expected SourceID 'test-source', got %s", firstEvent.SourceID)
		}
		if firstEvent.InfoBaseID != "test-infobase" {
			t.Errorf("expected InfoBaseID 'test-infobase', got %s", firstEvent.InfoBaseID)
		}
		if firstEvent.PayloadJSON == "" {
			t.Error("expected PayloadJSON to be non-empty")
		}
	}
}

func TestFileReader_DoesNotDuplicateSameJSONLInBuffer(t *testing.T) {
	tmpDir := t.TempDir()
	bufferPath := filepath.Join(tmpDir, "test.db")
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")

	jsonlContent := `{"Date":"2026-05-01T10:00:00","User":"admin","Computer":"PC01"}
{"Date":"2026-05-01T10:01:00","User":"user1","Computer":"PC02"}
{"Date":"2026-05-01T10:02:00","User":"user2","Computer":"PC03"}
`
	if err := os.WriteFile(jsonlPath, []byte(jsonlContent), 0o644); err != nil {
		t.Fatalf("write test jsonl file: %v", err)
	}

	store, err := buffer.Open(bufferPath)
	if err != nil {
		t.Fatalf("open buffer: %v", err)
	}
	defer store.Close()

	cfg := config.SourceConfig{
		SourceID:     "test-source",
		InfoBaseID:   "test-infobase",
		SourceNodeID: "test-node",
		FilePath:     jsonlPath,
		Mode:         "file",
	}

	for i := 0; i < 2; i++ {
		statusStore := status.New("test-service", cfg.SourceID, cfg.InfoBaseID)
		r := reader.New(cfg, store, statusStore)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := r.Run(ctx); err != nil {
			cancel()
			t.Fatalf("reader run %d: %v", i+1, err)
		}
		cancel()
	}

	pending, err := store.FetchPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("fetch pending: %v", err)
	}
	if len(pending) != 3 {
		t.Fatalf("expected same JSONL to stay at 3 buffered events, got %d", len(pending))
	}
}

func TestFileReader_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	bufferPath := filepath.Join(tmpDir, "test.db")

	store, err := buffer.Open(bufferPath)
	if err != nil {
		t.Fatalf("open buffer: %v", err)
	}
	defer store.Close()

	cfg := config.SourceConfig{
		SourceID:   "test-source",
		InfoBaseID: "test-infobase",
		FilePath:   "/nonexistent/path/events.jsonl",
		Mode:       "file",
	}

	statusStore := status.New("test-service", cfg.SourceID, cfg.InfoBaseID)
	r := reader.New(cfg, store, statusStore)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.Run(ctx); err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestFileReader_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	bufferPath := filepath.Join(tmpDir, "test.db")
	jsonlPath := filepath.Join(tmpDir, "events.jsonl")

	// Create a JSONL file with many events
	jsonlContent := ""
	for i := 0; i < 100; i++ {
		jsonlContent += `{"Date":"2026-05-01T10:00:00","User":"user","Computer":"PC"}` + "\n"
	}
	if err := os.WriteFile(jsonlPath, []byte(jsonlContent), 0o644); err != nil {
		t.Fatalf("write test jsonl file: %v", err)
	}

	store, err := buffer.Open(bufferPath)
	if err != nil {
		t.Fatalf("open buffer: %v", err)
	}
	defer store.Close()

	cfg := config.SourceConfig{
		SourceID:     "test-source",
		InfoBaseID:   "test-infobase",
		SourceNodeID: "test-node",
		FilePath:     jsonlPath,
		Mode:         "file",
	}

	statusStore := status.New("test-service", cfg.SourceID, cfg.InfoBaseID)
	r := reader.New(cfg, store, statusStore)

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err = r.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
