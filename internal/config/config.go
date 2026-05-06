package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Service    ServiceConfig    `yaml:"service"`
	Source     SourceConfig     `yaml:"source"`
	Buffer     BufferConfig     `yaml:"buffer"`
	ClickHouse ClickHouseConfig `yaml:"clickhouse"`
}

type ServiceConfig struct {
	Name     string `yaml:"name"`
	HTTPAddr string `yaml:"http_addr"`
}

type SourceConfig struct {
	SourceID         string `yaml:"source_id"`
	SourceName       string `yaml:"source_name"`
	InfoBaseID       string `yaml:"infobase_id"`
	InfoBaseName     string `yaml:"infobase_name"`
	SourceNodeID     string `yaml:"source_node_id"`
	Mode             string `yaml:"mode"` // "ibcmd" or "file"
	IbcmdPath        string `yaml:"ibcmd_path"`
	LogPath          string `yaml:"log_path"`
	FilePath         string `yaml:"file_path"`
	From             string `yaml:"from"`
	FollowIntervalMS int    `yaml:"follow_interval_ms"`
}

type BufferConfig struct {
	Path                   string `yaml:"path"`
	BatchSize              int    `yaml:"batch_size"`
	RetentionAfterAckHours int    `yaml:"retention_after_ack_hours"`
}

type ClickHouseConfig struct {
	Address  string `yaml:"address"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Table    string `yaml:"table"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Service.HTTPAddr == "" {
		c.Service.HTTPAddr = ":8080"
	}
	if c.Source.SourceID == "" {
		return errors.New("source.source_id is required")
	}
	if c.Source.InfoBaseID == "" {
		return errors.New("source.infobase_id is required")
	}
	if c.Source.Mode == "" {
		c.Source.Mode = "ibcmd"
	}
	if c.Source.Mode != "ibcmd" && c.Source.Mode != "file" {
		return errors.New("source.mode must be 'ibcmd' or 'file'")
	}
	if c.Source.Mode == "ibcmd" {
		if c.Source.IbcmdPath == "" {
			c.Source.IbcmdPath = "ibcmd"
		}
		if c.Source.LogPath == "" {
			return errors.New("source.log_path is required for ibcmd mode")
		}
	}
	if c.Source.Mode == "file" {
		if c.Source.FilePath == "" {
			return errors.New("source.file_path is required for file mode")
		}
	}
	if c.Source.FollowIntervalMS <= 0 {
		c.Source.FollowIntervalMS = 1000
	}
	if c.Buffer.Path == "" {
		c.Buffer.Path = "./data/agent.db"
	}
	if c.Buffer.BatchSize <= 0 {
		c.Buffer.BatchSize = 500
	}
	if c.ClickHouse.Address == "" {
		return errors.New("clickhouse.address is required")
	}
	if c.ClickHouse.Database == "" {
		c.ClickHouse.Database = "onec_eventlog"
	}
	if c.ClickHouse.Table == "" {
		c.ClickHouse.Table = "eventlog_events"
	}
	return nil
}
