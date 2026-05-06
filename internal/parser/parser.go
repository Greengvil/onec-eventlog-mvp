package parser

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
)

type EventRecord struct {
	EventFingerprint              string
	SourceID                      string
	SourceNodeID                  string
	InfoBaseID                    string
	InfoBaseName                  string
	EventTime                     time.Time
	Level                         string
	Application                   string
	ApplicationPresentation       string
	EventName                     string
	EventPresentation             string
	UserID                        string
	UserName                      string
	MetadataNames                 []string
	MetadataPresentations         []string
	Comment                       string
	DataPresentation              string
	DataJSON                      string
	TransactionStatus             string
	TransactionID                 string
	Connection                    int64
	Session                       int64
	ServerName                    string
	Port                          int64
	SyncPort                      int64
	SessionDataSeparationJSON     string
	SessionDataSeparationPresJSON string
	RawPayload                    string
	RawHash                       string
	ParserVersion                 string
	IngestedAt                    time.Time
}

func Parse(msg buffer.EventMessage) (EventRecord, error) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(msg.PayloadJSON), &obj); err != nil {
		return EventRecord{}, fmt.Errorf("parse raw eventlog json: %w", err)
	}

	eventTime := time.Now().UTC()
	if msg.EventTime.Valid {
		eventTime = msg.EventTime.Time.UTC()
	} else if value := stringValue(obj["Date"]); value != "" {
		if parsed := parseTime(value); parsed.Valid {
			eventTime = parsed.Time.UTC()
		}
	}

	dataJSON := jsonString(obj["Data"])
	sessionDataSeparationJSON := jsonString(obj["SessionDataSeparation"])
	sessionDataSeparationPresentationJSON := jsonString(obj["SessionDataSeparationPresentation"])

	record := EventRecord{
		SourceID:                      msg.SourceID,
		SourceNodeID:                  msg.SourceNodeID,
		InfoBaseID:                    msg.InfoBaseID,
		InfoBaseName:                  msg.InfoBaseName,
		EventTime:                     eventTime,
		Level:                         stringValue(obj["Level"]),
		Application:                   firstStringValue(obj, "Application", "ApplicationName"),
		ApplicationPresentation:       stringValue(obj["ApplicationPresentation"]),
		EventName:                     firstStringValue(obj, "EventName", "Event"),
		EventPresentation:             stringValue(obj["EventPresentation"]),
		UserID:                        firstStringValue(obj, "UserID", "User"),
		UserName:                      firstStringValue(obj, "UserName", "User"),
		MetadataNames:                 firstStringSlice(obj, "MetadataName", "Metadata"),
		MetadataPresentations:         stringSlice(obj["MetadataPresentation"]),
		Comment:                       stringValue(obj["Comment"]),
		DataPresentation:              stringValue(obj["DataPresentation"]),
		DataJSON:                      dataJSON,
		TransactionStatus:             stringValue(obj["TransactionStatus"]),
		TransactionID:                 stringValue(obj["TransactionID"]),
		Connection:                    intValue(obj["Connection"]),
		Session:                       intValue(obj["Session"]),
		ServerName:                    stringValue(obj["ServerName"]),
		Port:                          intValue(obj["Port"]),
		SyncPort:                      intValue(obj["SyncPort"]),
		SessionDataSeparationJSON:     sessionDataSeparationJSON,
		SessionDataSeparationPresJSON: sessionDataSeparationPresentationJSON,
		RawPayload:                    msg.PayloadJSON,
		RawHash:                       msg.PayloadHash,
		ParserVersion:                 "0.1.0",
		IngestedAt:                    time.Now().UTC(),
	}
	record.EventFingerprint = fingerprint(record)
	return record, nil
}

func firstStringValue(obj map[string]any, names ...string) string {
	for _, name := range names {
		if value := stringValue(obj[name]); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(v any) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		data, _ := json.Marshal(typed)
		return string(data)
	}
}

func firstStringSlice(obj map[string]any, names ...string) []string {
	for _, name := range names {
		if value := stringSlice(obj[name]); len(value) > 0 {
			return value
		}
	}
	return nil
}

func stringSlice(v any) []string {
	switch typed := v.(type) {
	case nil:
		return nil
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if s := stringValue(item); s != "" {
				result = append(result, s)
			}
		}
		return result
	default:
		return []string{stringValue(typed)}
	}
}

func intValue(v any) int64 {
	switch typed := v.(type) {
	case nil:
		return 0
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(typed, 10, 64)
		return parsed
	default:
		return 0
	}
}

func jsonString(v any) string {
	if v == nil {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseTime(value string) sql.NullTime {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return sql.NullTime{Time: parsed, Valid: true}
		}
	}
	return sql.NullTime{}
}

func fingerprint(record EventRecord) string {
	base := strings.Join([]string{
		record.SourceID,
		record.EventTime.Format(time.RFC3339Nano),
		record.EventName,
		record.UserID,
		record.UserName,
		strings.Join(record.MetadataNames, ","),
		record.TransactionID,
		strconv.FormatInt(record.Connection, 10),
		strconv.FormatInt(record.Session, 10),
		record.RawHash,
	}, "|")
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])
}
