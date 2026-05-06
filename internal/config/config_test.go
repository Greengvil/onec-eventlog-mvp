package config

import "testing"

func TestValidateDefaultsHTTPAddrToLoopback(t *testing.T) {
	cfg := Config{
		Source: SourceConfig{
			SourceID:   "source-1",
			InfoBaseID: "ib-1",
			Mode:       "file",
			FilePath:   "./events.jsonl",
		},
		ClickHouse: ClickHouseConfig{
			Address: "localhost:9000",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	if cfg.Service.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("HTTPAddr = %q, want loopback default", cfg.Service.HTTPAddr)
	}
}

func TestValidateKeepsExplicitHTTPAddr(t *testing.T) {
	cfg := Config{
		Service: ServiceConfig{
			HTTPAddr: "0.0.0.0:8080",
		},
		Source: SourceConfig{
			SourceID:   "source-1",
			InfoBaseID: "ib-1",
			Mode:       "file",
			FilePath:   "./events.jsonl",
		},
		ClickHouse: ClickHouseConfig{
			Address: "localhost:9000",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	if cfg.Service.HTTPAddr != "0.0.0.0:8080" {
		t.Fatalf("HTTPAddr = %q, want explicit address preserved", cfg.Service.HTTPAddr)
	}
}
