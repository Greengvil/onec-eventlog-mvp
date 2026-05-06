package clickhouse

import (
	"context"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/example/onec-eventlog-mvp/internal/config"
	"github.com/example/onec-eventlog-mvp/internal/parser"
)

type Writer struct {
	conn  ch.Conn
	table string
}

func Open(ctx context.Context, cfg config.ClickHouseConfig) (*Writer, error) {
	conn, err := ch.Open(&ch.Options{
		Addr: []string{cfg.Address},
		Auth: ch.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}
	return &Writer{conn: conn, table: cfg.Database + "." + cfg.Table}, nil
}

func (w *Writer) Insert(ctx context.Context, records []parser.EventRecord) error {
	if len(records) == 0 {
		return nil
	}
	query := fmt.Sprintf(`INSERT INTO %s (
		event_fingerprint, source_id, source_node_id, infobase_id, infobase_name,
		event_time, level, application, application_presentation, event_name, event_presentation,
		user_id, user_name, metadata_names, metadata_presentations, comment, data_presentation,
		data_json, transaction_status, transaction_id, connection, session, server_name,
		port, sync_port, session_data_separation_json, session_data_separation_presentation_json,
		raw_payload, raw_hash, parser_version, ingested_at
	)`, w.table)

	batch, err := w.conn.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare clickhouse batch: %w", err)
	}
	for _, record := range records {
		if err := batch.Append(
			record.EventFingerprint,
			record.SourceID,
			record.SourceNodeID,
			record.InfoBaseID,
			record.InfoBaseName,
			record.EventTime,
			record.Level,
			record.Application,
			record.ApplicationPresentation,
			record.EventName,
			record.EventPresentation,
			record.UserID,
			record.UserName,
			record.MetadataNames,
			record.MetadataPresentations,
			record.Comment,
			record.DataPresentation,
			record.DataJSON,
			record.TransactionStatus,
			record.TransactionID,
			record.Connection,
			record.Session,
			record.ServerName,
			record.Port,
			record.SyncPort,
			record.SessionDataSeparationJSON,
			record.SessionDataSeparationPresJSON,
			record.RawPayload,
			record.RawHash,
			record.ParserVersion,
			record.IngestedAt,
		); err != nil {
			return fmt.Errorf("append clickhouse record: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send clickhouse batch: %w", err)
	}
	return nil
}
