package clickhouse

import (
	"net/url"
	"testing"
)

func TestParseEventSearchParamsDefaultLimit(t *testing.T) {
	params, err := ParseEventSearchParams(url.Values{})
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}
	if params.Limit != defaultEventsLimit {
		t.Fatalf("Limit = %d, want %d", params.Limit, defaultEventsLimit)
	}
}

func TestParseEventSearchParamsCapsLimit(t *testing.T) {
	params, err := ParseEventSearchParams(url.Values{"limit": {"5000"}})
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}
	if params.Limit != maxEventsLimit {
		t.Fatalf("Limit = %d, want %d", params.Limit, maxEventsLimit)
	}
}

func TestParseEventSearchParamsRejectsInvalidLimit(t *testing.T) {
	if _, err := ParseEventSearchParams(url.Values{"limit": {"bad"}}); err == nil {
		t.Fatal("expected invalid limit error")
	}
	if _, err := ParseEventSearchParams(url.Values{"limit": {"0"}}); err == nil {
		t.Fatal("expected non-positive limit error")
	}
}

func TestParseEventSearchParamsFilters(t *testing.T) {
	params, err := ParseEventSearchParams(url.Values{
		"source_id":  {"real-jr-local"},
		"from":       {"2026-03-04T16:41:37Z"},
		"to":         {"2026-03-18"},
		"event_name": {"_$Session$_.Start"},
		"user_name":  {"user"},
		"metadata":   {"Catalog.Products"},
		"limit":      {"5"},
	})
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}
	if params.SourceID != "real-jr-local" || params.EventName != "_$Session$_.Start" || params.UserName != "user" || params.Metadata != "Catalog.Products" || params.Limit != 5 {
		t.Fatalf("unexpected params: %#v", params)
	}
	if params.From.IsZero() || params.To.IsZero() {
		t.Fatalf("expected parsed from/to: %#v", params)
	}
}
