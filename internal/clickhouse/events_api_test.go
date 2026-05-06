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

func TestEventFingerprintFromPath(t *testing.T) {
	fingerprint, err := EventFingerprintFromPath("/api/events/abcdef123")
	if err != nil {
		t.Fatalf("parse fingerprint: %v", err)
	}
	if fingerprint != "abcdef123" {
		t.Fatalf("fingerprint = %q", fingerprint)
	}
}

func TestEventFingerprintFromPathRejectsInvalidPath(t *testing.T) {
	if _, err := EventFingerprintFromPath("/api/events/"); err == nil {
		t.Fatal("expected empty fingerprint error")
	}
	if _, err := EventFingerprintFromPath("/api/events/a/b"); err == nil {
		t.Fatal("expected slash error")
	}
}

func TestEventFingerprintFromNeighborsPath(t *testing.T) {
	fingerprint, err := EventFingerprintFromNeighborsPath("/api/events/abcdef123/neighbors")
	if err != nil {
		t.Fatalf("parse fingerprint: %v", err)
	}
	if fingerprint != "abcdef123" {
		t.Fatalf("fingerprint = %q", fingerprint)
	}
}

func TestParseNeighborLimit(t *testing.T) {
	limit, err := ParseNeighborLimit(url.Values{})
	if err != nil {
		t.Fatalf("parse default limit: %v", err)
	}
	if limit != defaultNeighborsLimit {
		t.Fatalf("limit = %d, want %d", limit, defaultNeighborsLimit)
	}

	limit, err = ParseNeighborLimit(url.Values{"limit": {"500"}})
	if err != nil {
		t.Fatalf("parse capped limit: %v", err)
	}
	if limit != maxNeighborsLimit {
		t.Fatalf("limit = %d, want %d", limit, maxNeighborsLimit)
	}

	limit, err = ParseNeighborLimit(url.Values{"limit": {"3"}})
	if err != nil {
		t.Fatalf("parse explicit limit: %v", err)
	}
	if limit != 3 {
		t.Fatalf("limit = %d, want 3", limit)
	}
}

func TestParseNeighborLimitRejectsInvalidLimit(t *testing.T) {
	if _, err := ParseNeighborLimit(url.Values{"limit": {"bad"}}); err == nil {
		t.Fatal("expected invalid limit error")
	}
	if _, err := ParseNeighborLimit(url.Values{"limit": {"0"}}); err == nil {
		t.Fatal("expected non-positive limit error")
	}
}
