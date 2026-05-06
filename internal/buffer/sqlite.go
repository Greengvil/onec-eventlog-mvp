package buffer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type EventMessage struct {
	LocalID        int64
	MessageID      string
	SourceID       string
	SourceNodeID   string
	InfoBaseID     string
	InfoBaseName   string
	Seq            int64
	EventTime      sql.NullTime
	PayloadHash    string
	PayloadJSON    string
	Status         string
	Attempts       int
	LastError      sql.NullString
	CreatedAt      time.Time
	AcknowledgedAt sql.NullTime
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create buffer directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite buffer: %w", err)
	}
	store := &Store{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) init() error {
	queries := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=NORMAL;`,
		`CREATE TABLE IF NOT EXISTS local_event_buffer (
			local_id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id TEXT NOT NULL UNIQUE,
			source_id TEXT NOT NULL,
			source_node_id TEXT NOT NULL,
			infobase_id TEXT NOT NULL,
			infobase_name TEXT,
			seq INTEGER NOT NULL,
			event_time TEXT,
			payload_hash TEXT NOT NULL,
			payload_json TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TEXT NOT NULL,
			acknowledged_at TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_local_event_buffer_status ON local_event_buffer(status, local_id);`,
		`CREATE INDEX IF NOT EXISTS idx_local_event_buffer_source_seq ON local_event_buffer(source_id, seq);`,
		`CREATE TABLE IF NOT EXISTS reader_state (
			source_id TEXT PRIMARY KEY,
			last_seq INTEGER NOT NULL DEFAULT 0,
			last_event_time TEXT,
			status TEXT NOT NULL DEFAULT 'starting',
			last_error TEXT,
			updated_at TEXT NOT NULL
		);`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("init sqlite buffer: %w", err)
		}
	}
	return nil
}

func (s *Store) Put(ctx context.Context, msg EventMessage) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO local_event_buffer (
			message_id, source_id, source_node_id, infobase_id, infobase_name,
			seq, event_time, payload_hash, payload_json, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)
	`, msg.MessageID, msg.SourceID, msg.SourceNodeID, msg.InfoBaseID, msg.InfoBaseName,
		msg.Seq, nullableTimeString(msg.EventTime), msg.PayloadHash, msg.PayloadJSON, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("put event to sqlite buffer: %w", err)
	}
	return nil
}

func (s *Store) FetchPending(ctx context.Context, limit int) ([]EventMessage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT local_id, message_id, source_id, source_node_id, infobase_id, infobase_name,
		       seq, event_time, payload_hash, payload_json, status, attempts, last_error, created_at, acknowledged_at
		FROM local_event_buffer
		WHERE status IN ('pending', 'retry')
		ORDER BY local_id
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending events: %w", err)
	}
	defer rows.Close()

	var result []EventMessage
	for rows.Next() {
		var msg EventMessage
		var eventTime, createdAt, acknowledgedAt sql.NullString
		if err := rows.Scan(&msg.LocalID, &msg.MessageID, &msg.SourceID, &msg.SourceNodeID, &msg.InfoBaseID, &msg.InfoBaseName,
			&msg.Seq, &eventTime, &msg.PayloadHash, &msg.PayloadJSON, &msg.Status, &msg.Attempts, &msg.LastError, &createdAt, &acknowledgedAt); err != nil {
			return nil, fmt.Errorf("scan pending event: %w", err)
		}
		msg.EventTime = parseNullableTime(eventTime)
		msg.CreatedAt = parseTimeOrNow(createdAt)
		msg.AcknowledgedAt = parseNullableTime(acknowledgedAt)
		result = append(result, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending events: %w", err)
	}
	return result, nil
}

func (s *Store) MarkDone(ctx context.Context, localIDs []int64) error {
	if len(localIDs) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE local_event_buffer SET status='acknowledged', acknowledged_at=? WHERE local_id=?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, id := range localIDs {
		if _, err := stmt.ExecContext(ctx, now, id); err != nil {
			return fmt.Errorf("mark event acknowledged: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) MarkError(ctx context.Context, localID int64, errText string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE local_event_buffer
		SET status = CASE WHEN attempts >= 5 THEN 'dead' ELSE 'retry' END,
		    attempts = attempts + 1,
		    last_error = ?
		WHERE local_id = ?
	`, errText, localID)
	return err
}

func (s *Store) PendingCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM local_event_buffer WHERE status IN ('pending','retry')`).Scan(&count)
	return count, err
}

func nullableTimeString(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return t.Time.UTC().Format(time.RFC3339Nano)
}

func parseNullableTime(value sql.NullString) sql.NullTime {
	if !value.Valid || value.String == "" {
		return sql.NullTime{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: parsed, Valid: true}
}

func parseTimeOrNow(value sql.NullString) time.Time {
	if !value.Valid || value.String == "" {
		return time.Now().UTC()
	}
	parsed, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return time.Now().UTC()
	}
	return parsed
}
