package reader

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
	"github.com/example/onec-eventlog-mvp/internal/config"
	"github.com/example/onec-eventlog-mvp/internal/status"
)

// FileReader reads events from a JSONL file (one event per line).
type FileReader struct {
	cfg    config.SourceConfig
	store  *buffer.Store
	status *status.Store
}

// NewFileReader creates a new FileReader.
func NewFileReader(cfg config.SourceConfig, store *buffer.Store, status *status.Store) *FileReader {
	return &FileReader{cfg: cfg, store: store, status: status}
}

// Run reads events from the configured file and stores them in the buffer.
func (r *FileReader) Run(ctx context.Context) error {
	file, err := os.Open(r.cfg.FilePath)
	if err != nil {
		return fmt.Errorf("open file %s: %w", r.cfg.FilePath, err)
	}
	defer file.Close()

	r.status.SetStatus("running")
	scanner := bufio.NewScanner(file)
	var seq int64

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		// Validate that the line is valid JSON
		var raw json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			r.status.SetError(fmt.Errorf("invalid json at line: %s", err))
			continue
		}

		seq++
		r.status.IncRead()

		payloadHash := sha256Hex(line)
		messageID := messageID(r.cfg.SourceID, payloadHash)
		eventTime := extractEventTime(line)

		msg := buffer.EventMessage{
			MessageID:    messageID,
			SourceID:     r.cfg.SourceID,
			SourceNodeID: r.cfg.SourceNodeID,
			InfoBaseID:   r.cfg.InfoBaseID,
			InfoBaseName: r.cfg.InfoBaseName,
			Seq:          seq,
			EventTime:    eventTime,
			PayloadHash:  "sha256:" + payloadHash,
			PayloadJSON:  string(line),
		}
		if err := r.store.Put(ctx, msg); err != nil {
			return err
		}
		r.status.IncBuffered()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read file %s: %w", r.cfg.FilePath, err)
	}

	return nil
}
