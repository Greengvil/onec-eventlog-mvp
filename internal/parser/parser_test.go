package parser_test

import (
	"slices"
	"testing"
	"time"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
	"github.com/example/onec-eventlog-mvp/internal/parser"
)

func TestParseNormalizesEventLogFields(t *testing.T) {
	raw := `{"Date":"2026-05-01T10:01:02Z","Level":"W","Application":"1CV8","ApplicationPresentation":"1C:Enterprise","EventName":"Data.Update","EventPresentation":"Data update","UserID":"ivanov","UserName":"Ivanov I.I.","MetadataName":["Catalog.Products","Document.SalesOrder"],"MetadataPresentation":["Products","Sales order"],"Comment":"Updated smoke data","DataPresentation":"Catalog.Products: Smoke item","Data":{"ref":"product-1"},"TransactionStatus":"Commit","TransactionID":"tx-001","Connection":42,"Session":77,"ServerName":"app01"}`

	record, err := parser.Parse(buffer.EventMessage{
		SourceID:     "source-1",
		SourceNodeID: "node-1",
		InfoBaseID:   "ib-1",
		InfoBaseName: "Infobase 1",
		PayloadHash:  "sha256:test",
		PayloadJSON:  raw,
	})
	if err != nil {
		t.Fatalf("parse event: %v", err)
	}

	wantTime := time.Date(2026, 5, 1, 10, 1, 2, 0, time.UTC)
	if !record.EventTime.Equal(wantTime) {
		t.Fatalf("EventTime = %s, want %s", record.EventTime, wantTime)
	}

	if record.Level != "W" {
		t.Errorf("Level = %q", record.Level)
	}
	if record.Application != "1CV8" {
		t.Errorf("Application = %q", record.Application)
	}
	if record.ApplicationPresentation != "1C:Enterprise" {
		t.Errorf("ApplicationPresentation = %q", record.ApplicationPresentation)
	}
	if record.EventName != "Data.Update" {
		t.Errorf("EventName = %q", record.EventName)
	}
	if record.EventPresentation != "Data update" {
		t.Errorf("EventPresentation = %q", record.EventPresentation)
	}
	if record.UserID != "ivanov" {
		t.Errorf("UserID = %q", record.UserID)
	}
	if record.UserName != "Ivanov I.I." {
		t.Errorf("UserName = %q", record.UserName)
	}
	if !slices.Equal(record.MetadataNames, []string{"Catalog.Products", "Document.SalesOrder"}) {
		t.Errorf("MetadataNames = %#v", record.MetadataNames)
	}
	if !slices.Equal(record.MetadataPresentations, []string{"Products", "Sales order"}) {
		t.Errorf("MetadataPresentations = %#v", record.MetadataPresentations)
	}
	if record.Comment != "Updated smoke data" {
		t.Errorf("Comment = %q", record.Comment)
	}
	if record.DataPresentation != "Catalog.Products: Smoke item" {
		t.Errorf("DataPresentation = %q", record.DataPresentation)
	}
	if record.DataJSON != `{"ref":"product-1"}` {
		t.Errorf("DataJSON = %q", record.DataJSON)
	}
	if record.TransactionStatus != "Commit" {
		t.Errorf("TransactionStatus = %q", record.TransactionStatus)
	}
	if record.TransactionID != "tx-001" {
		t.Errorf("TransactionID = %q", record.TransactionID)
	}
	if record.Connection != 42 {
		t.Errorf("Connection = %d", record.Connection)
	}
	if record.Session != 77 {
		t.Errorf("Session = %d", record.Session)
	}
	if record.ServerName != "app01" {
		t.Errorf("ServerName = %q", record.ServerName)
	}
	if record.RawPayload != raw {
		t.Errorf("RawPayload = %q", record.RawPayload)
	}
	if record.RawHash != "sha256:test" {
		t.Errorf("RawHash = %q", record.RawHash)
	}
	if record.EventFingerprint == "" {
		t.Error("EventFingerprint is empty")
	}
}

func TestParseSupportsIbcmdEventLogAliases(t *testing.T) {
	raw := `{"ApplicationName":"Designer","ApplicationPresentation":"Конфигуратор","Comment":"","Connection":"1","DataPresentation":"alias smoke","Date":"2026-03-04T16:41:37","Event":"_$Session$_.Authentication","EventPresentation":"Сеанс. Аутентификация","Level":"Information","Metadata":"00000000-0000-0000-0000-000000000000","MetadataPresentation":"<Не определено>","ServerName":"app01","Session":"1","TransactionID":"tx-real","TransactionStatus":"NotApplicable","User":"071523a4-516f-4fce-ba4b-0d11ab7a1893","UserName":""}`

	record, err := parser.Parse(buffer.EventMessage{
		SourceID:    "real-source",
		PayloadHash: "sha256:real",
		PayloadJSON: raw,
	})
	if err != nil {
		t.Fatalf("parse event: %v", err)
	}

	if record.Application != "Designer" {
		t.Errorf("Application = %q", record.Application)
	}
	if record.ApplicationPresentation != "Конфигуратор" {
		t.Errorf("ApplicationPresentation = %q", record.ApplicationPresentation)
	}
	if record.EventName != "_$Session$_.Authentication" {
		t.Errorf("EventName = %q", record.EventName)
	}
	if record.EventPresentation != "Сеанс. Аутентификация" {
		t.Errorf("EventPresentation = %q", record.EventPresentation)
	}
	if record.UserID != "071523a4-516f-4fce-ba4b-0d11ab7a1893" {
		t.Errorf("UserID = %q", record.UserID)
	}
	if record.UserName != "071523a4-516f-4fce-ba4b-0d11ab7a1893" {
		t.Errorf("UserName = %q", record.UserName)
	}
	if !slices.Equal(record.MetadataNames, []string{"00000000-0000-0000-0000-000000000000"}) {
		t.Errorf("MetadataNames = %#v", record.MetadataNames)
	}
	if !slices.Equal(record.MetadataPresentations, []string{"<Не определено>"}) {
		t.Errorf("MetadataPresentations = %#v", record.MetadataPresentations)
	}
	if record.DataPresentation != "alias smoke" {
		t.Errorf("DataPresentation = %q", record.DataPresentation)
	}
	if record.TransactionStatus != "NotApplicable" {
		t.Errorf("TransactionStatus = %q", record.TransactionStatus)
	}
	if record.TransactionID != "tx-real" {
		t.Errorf("TransactionID = %q", record.TransactionID)
	}
	if record.Connection != 1 {
		t.Errorf("Connection = %d", record.Connection)
	}
	if record.Session != 1 {
		t.Errorf("Session = %d", record.Session)
	}
	if record.ServerName != "app01" {
		t.Errorf("ServerName = %q", record.ServerName)
	}
}

func TestParseEventFingerprintIsStableForSameSourceAndRawPayload(t *testing.T) {
	raw := `{"Date":"2026-05-01T10:01:02Z","EventName":"Data.Update","UserID":"ivanov","UserName":"Ivanov I.I.","MetadataName":["Catalog.Products"],"TransactionID":"tx-001","Connection":42,"Session":77}`
	msg := buffer.EventMessage{
		SourceID:    "source-1",
		PayloadHash: "sha256:same-raw",
		PayloadJSON: raw,
	}

	first, err := parser.Parse(msg)
	if err != nil {
		t.Fatalf("parse first event: %v", err)
	}
	second, err := parser.Parse(msg)
	if err != nil {
		t.Fatalf("parse second event: %v", err)
	}

	if first.EventFingerprint == "" {
		t.Fatal("EventFingerprint is empty")
	}
	if first.EventFingerprint != second.EventFingerprint {
		t.Fatalf("same source and raw payload produced different fingerprints: %q != %q", first.EventFingerprint, second.EventFingerprint)
	}
}

func TestParseEventFingerprintDiffersForDifferentSource(t *testing.T) {
	raw := `{"Date":"2026-05-01T10:01:02Z","EventName":"Data.Update","UserID":"ivanov","UserName":"Ivanov I.I.","MetadataName":["Catalog.Products"],"TransactionID":"tx-001","Connection":42,"Session":77}`

	first, err := parser.Parse(buffer.EventMessage{
		SourceID:    "source-1",
		PayloadHash: "sha256:same-raw",
		PayloadJSON: raw,
	})
	if err != nil {
		t.Fatalf("parse first event: %v", err)
	}
	second, err := parser.Parse(buffer.EventMessage{
		SourceID:    "source-2",
		PayloadHash: "sha256:same-raw",
		PayloadJSON: raw,
	})
	if err != nil {
		t.Fatalf("parse second event: %v", err)
	}

	if first.EventFingerprint == second.EventFingerprint {
		t.Fatalf("different sources produced same fingerprint: %q", first.EventFingerprint)
	}
}
