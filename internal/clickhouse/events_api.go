package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEventsLimit = 100
	maxEventsLimit     = 1000
)

type EventSearchParams struct {
	SourceID  string
	From      time.Time
	To        time.Time
	EventName string
	UserName  string
	Metadata  string
	Limit     int
}

type EventSearchResult struct {
	EventTime         time.Time `json:"event_time"`
	Level             string    `json:"level"`
	Application       string    `json:"application"`
	EventName         string    `json:"event_name"`
	UserID            string    `json:"user_id"`
	UserName          string    `json:"user_name"`
	MetadataNames     []string  `json:"metadata_names"`
	DataPresentation  string    `json:"data_presentation"`
	TransactionStatus string    `json:"transaction_status"`
	TransactionID     string    `json:"transaction_id"`
	Connection        int64     `json:"connection"`
	Session           int64     `json:"session"`
	ServerName        string    `json:"server_name"`
	EventFingerprint  string    `json:"event_fingerprint"`
}

type EventDetailResult struct {
	EventTime               time.Time `json:"event_time"`
	Level                   string    `json:"level"`
	Application             string    `json:"application"`
	ApplicationPresentation string    `json:"application_presentation"`
	EventName               string    `json:"event_name"`
	EventPresentation       string    `json:"event_presentation"`
	UserID                  string    `json:"user_id"`
	UserName                string    `json:"user_name"`
	MetadataNames           []string  `json:"metadata_names"`
	MetadataPresentations   []string  `json:"metadata_presentations"`
	Comment                 string    `json:"comment"`
	DataPresentation        string    `json:"data_presentation"`
	TransactionStatus       string    `json:"transaction_status"`
	TransactionID           string    `json:"transaction_id"`
	Connection              int64     `json:"connection"`
	Session                 int64     `json:"session"`
	ServerName              string    `json:"server_name"`
	EventFingerprint        string    `json:"event_fingerprint"`
	RawPayload              string    `json:"raw_payload"`
	IngestedAt              time.Time `json:"ingested_at"`
}

func (w *Writer) EventsHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		params, err := ParseEventSearchParams(r.URL.Query())
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		events, err := w.SearchEvents(r.Context(), params)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(events)
	}
}

func (w *Writer) EventDetailHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		fingerprint, err := EventFingerprintFromPath(r.URL.Path)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		event, found, err := w.GetEventByFingerprint(r.Context(), fingerprint)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(rw, r)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(event)
	}
}

func EventFingerprintFromPath(path string) (string, error) {
	fingerprint := strings.TrimPrefix(path, "/api/events/")
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" || fingerprint == path {
		return "", fmt.Errorf("event_fingerprint is required")
	}
	if strings.Contains(fingerprint, "/") {
		return "", fmt.Errorf("event_fingerprint must not contain slash")
	}
	return fingerprint, nil
}

func ParseEventSearchParams(values url.Values) (EventSearchParams, error) {
	params := EventSearchParams{Limit: defaultEventsLimit}
	params.SourceID = values.Get("source_id")
	params.EventName = values.Get("event_name")
	params.UserName = values.Get("user_name")
	params.Metadata = values.Get("metadata")

	if raw := values.Get("from"); raw != "" {
		parsed, err := parseSearchTime(raw)
		if err != nil {
			return EventSearchParams{}, fmt.Errorf("invalid from: %w", err)
		}
		params.From = parsed
	}
	if raw := values.Get("to"); raw != "" {
		parsed, err := parseSearchTime(raw)
		if err != nil {
			return EventSearchParams{}, fmt.Errorf("invalid to: %w", err)
		}
		params.To = parsed
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return EventSearchParams{}, fmt.Errorf("invalid limit: %w", err)
		}
		if limit <= 0 {
			return EventSearchParams{}, fmt.Errorf("limit must be positive")
		}
		if limit > maxEventsLimit {
			limit = maxEventsLimit
		}
		params.Limit = limit
	}

	return params, nil
}

func parseSearchTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DDTHH:MM:SS")
}

func (w *Writer) SearchEvents(ctx context.Context, params EventSearchParams) ([]EventSearchResult, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 8)

	if params.SourceID != "" {
		where = append(where, "source_id = ?")
		args = append(args, params.SourceID)
	}
	if !params.From.IsZero() {
		where = append(where, "event_time >= ?")
		args = append(args, params.From)
	}
	if !params.To.IsZero() {
		where = append(where, "event_time <= ?")
		args = append(args, params.To)
	}
	if params.EventName != "" {
		where = append(where, "event_name = ?")
		args = append(args, params.EventName)
	}
	if params.UserName != "" {
		where = append(where, "user_name = ?")
		args = append(args, params.UserName)
	}
	if params.Metadata != "" {
		where = append(where, "has(metadata_names, ?)")
		args = append(args, params.Metadata)
	}

	query := fmt.Sprintf(`
		SELECT
			event_time, level, application, event_name, user_id, user_name, metadata_names,
			data_presentation, transaction_status, transaction_id, connection, session,
			server_name, event_fingerprint
		FROM (
			SELECT
				event_time, level, application, event_name, user_id, user_name, metadata_names,
				data_presentation, transaction_status, transaction_id, connection, session,
				server_name, event_fingerprint, ingested_at,
				row_number() OVER (PARTITION BY event_fingerprint ORDER BY ingested_at DESC) AS rn
			FROM %s
			WHERE %s
		)
		WHERE rn = 1
		ORDER BY event_time DESC
		LIMIT ?
	`, w.table, strings.Join(where, " AND "))
	args = append(args, params.Limit)

	rows, err := w.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var result []EventSearchResult
	for rows.Next() {
		var event EventSearchResult
		if err := rows.Scan(
			&event.EventTime,
			&event.Level,
			&event.Application,
			&event.EventName,
			&event.UserID,
			&event.UserName,
			&event.MetadataNames,
			&event.DataPresentation,
			&event.TransactionStatus,
			&event.TransactionID,
			&event.Connection,
			&event.Session,
			&event.ServerName,
			&event.EventFingerprint,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return result, nil
}

func (w *Writer) GetEventByFingerprint(ctx context.Context, fingerprint string) (EventDetailResult, bool, error) {
	query := fmt.Sprintf(`
		SELECT
			event_time, level, application, application_presentation, event_name, event_presentation,
			user_id, user_name, metadata_names, metadata_presentations, comment, data_presentation,
			transaction_status, transaction_id, connection, session, server_name, event_fingerprint,
			raw_payload, ingested_at
		FROM %s
		WHERE event_fingerprint = ?
		ORDER BY ingested_at DESC
		LIMIT 1
	`, w.table)

	rows, err := w.conn.Query(ctx, query, fingerprint)
	if err != nil {
		return EventDetailResult{}, false, fmt.Errorf("query event detail: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return EventDetailResult{}, false, fmt.Errorf("iterate event detail: %w", err)
		}
		return EventDetailResult{}, false, nil
	}

	var event EventDetailResult
	if err := rows.Scan(
		&event.EventTime,
		&event.Level,
		&event.Application,
		&event.ApplicationPresentation,
		&event.EventName,
		&event.EventPresentation,
		&event.UserID,
		&event.UserName,
		&event.MetadataNames,
		&event.MetadataPresentations,
		&event.Comment,
		&event.DataPresentation,
		&event.TransactionStatus,
		&event.TransactionID,
		&event.Connection,
		&event.Session,
		&event.ServerName,
		&event.EventFingerprint,
		&event.RawPayload,
		&event.IngestedAt,
	); err != nil {
		return EventDetailResult{}, false, fmt.Errorf("scan event detail: %w", err)
	}
	return event, true, nil
}
