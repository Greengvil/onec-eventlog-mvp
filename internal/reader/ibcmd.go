package reader

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
	"github.com/example/onec-eventlog-mvp/internal/config"
	"github.com/example/onec-eventlog-mvp/internal/status"
)

type IbcmdReader struct {
	cfg    config.SourceConfig
	store  *buffer.Store
	status *status.Store
}

func NewIbcmdReader(cfg config.SourceConfig, store *buffer.Store, status *status.Store) *IbcmdReader {
	return &IbcmdReader{cfg: cfg, store: store, status: status}
}

func (r *IbcmdReader) Run(ctx context.Context) error {
	args := []string{
		"eventlog", "export",
		"--format=json",
		"--skip-root",
	}
	if r.cfg.From != "" {
		args = append(args, "--from="+r.cfg.From)
	}
	args = append(args, "--follow="+strconv.Itoa(r.cfg.FollowIntervalMS))
	args = append(args, r.cfg.LogPath)

	cmd := exec.CommandContext(ctx, r.cfg.IbcmdPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open ibcmd stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("open ibcmd stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ibcmd: %w", err)
	}

	r.status.SetStatus("running")
	go r.readStderr(stderr)

	decodeErr := r.decodeStream(ctx, stdout)
	waitErr := cmd.Wait()
	if decodeErr != nil && ctx.Err() == nil {
		return decodeErr
	}
	if waitErr != nil && ctx.Err() == nil {
		return fmt.Errorf("ibcmd stopped with error: %w", waitErr)
	}
	return ctx.Err()
}

func (r *IbcmdReader) decodeStream(ctx context.Context, stream io.Reader) error {
	// ibcmd с --skip-root удобнее читать как последовательность JSON-объектов.
	// Если конкретная версия платформы отдаёт объекты построчно, json.Decoder также корректно их прочитает.
	decoder := json.NewDecoder(bufio.NewReader(stream))
	var seq int64

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decode eventlog json from ibcmd: %w", err)
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}

		seq++
		r.status.IncRead()

		payloadHash := sha256Hex(raw)
		messageID := messageID(r.cfg.SourceID, payloadHash)
		eventTime := extractEventTime(raw)

		msg := buffer.EventMessage{
			MessageID:    messageID,
			SourceID:     r.cfg.SourceID,
			SourceNodeID: r.cfg.SourceNodeID,
			InfoBaseID:   r.cfg.InfoBaseID,
			InfoBaseName: r.cfg.InfoBaseName,
			Seq:          seq,
			EventTime:    eventTime,
			PayloadHash:  "sha256:" + payloadHash,
			PayloadJSON:  string(raw),
		}
		if err := r.store.Put(ctx, msg); err != nil {
			return err
		}
		r.status.IncBuffered()
	}
}

func (r *IbcmdReader) readStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			r.status.SetError(fmt.Errorf("ibcmd stderr: %s", line))
		}
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func messageID(sourceID, payloadHash string) string {
	base := fmt.Sprintf("%s|%s", sourceID, payloadHash)
	return sha256Hex([]byte(base))
}

func extractEventTime(raw []byte) sql.NullTime {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return sql.NullTime{}
	}
	value, ok := obj["Date"].(string)
	if !ok || value == "" {
		return sql.NullTime{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return sql.NullTime{Time: parsed.UTC(), Valid: true}
		}
	}
	return sql.NullTime{}
}
