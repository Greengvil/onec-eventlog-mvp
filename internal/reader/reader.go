package reader

import (
	"context"

	"github.com/example/onec-eventlog-mvp/internal/buffer"
	"github.com/example/onec-eventlog-mvp/internal/config"
	"github.com/example/onec-eventlog-mvp/internal/status"
)

// EventReader defines the interface for reading events from different sources.
type EventReader interface {
	Run(ctx context.Context) error
}

// New creates an appropriate event reader based on the configuration mode.
func New(cfg config.SourceConfig, store *buffer.Store, status *status.Store) EventReader {
	switch cfg.Mode {
	case "file":
		return NewFileReader(cfg, store, status)
	default: // "ibcmd" is the default
		return NewIbcmdReader(cfg, store, status)
	}
}
