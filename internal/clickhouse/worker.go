package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
	"github.com/example/onec-eventlog-mvp/internal/parser"
	"github.com/example/onec-eventlog-mvp/internal/status"
)

type Worker struct {
	buffer    *buffer.Store
	writer    *Writer
	batchSize int
	status    *status.Store
}

func NewWorker(buffer *buffer.Store, writer *Writer, batchSize int, status *status.Store) *Worker {
	return &Worker{buffer: buffer, writer: writer, batchSize: batchSize, status: status}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.processOnce(ctx); err != nil {
				w.status.SetError(err)
			}
		}
	}
}

func (w *Worker) processOnce(ctx context.Context) error {
	messages, err := w.buffer.FetchPending(ctx, w.batchSize)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}

	records := make([]parser.EventRecord, 0, len(messages))
	processedLocalIDs := make([]int64, 0, len(messages))

	for _, msg := range messages {
		record, err := parser.Parse(msg)
		if err != nil {
			w.status.IncFailed()
			_ = w.buffer.MarkError(ctx, msg.LocalID, err.Error())
			continue
		}
		records = append(records, record)
		processedLocalIDs = append(processedLocalIDs, msg.LocalID)
	}
	if len(records) == 0 {
		return nil
	}

	if err := w.writer.Insert(ctx, records); err != nil {
		for _, id := range processedLocalIDs {
			_ = w.buffer.MarkError(ctx, id, fmt.Sprintf("clickhouse insert: %v", err))
		}
		return err
	}

	if err := w.buffer.MarkDone(ctx, processedLocalIDs); err != nil {
		return err
	}

	w.status.IncParsed(int64(len(records)))
	w.status.IncWritten(int64(len(records)))
	return nil
}
