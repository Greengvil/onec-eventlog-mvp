package status

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type State struct {
	ServiceName       string    `json:"service_name"`
	Status            string    `json:"status"`
	StartedAt         time.Time `json:"started_at"`
	LastEventAt       time.Time `json:"last_event_at,omitempty"`
	ReadEvents        int64     `json:"read_events"`
	BufferedEvents    int64     `json:"buffered_events"`
	ParsedEvents      int64     `json:"parsed_events"`
	ClickHouseWrites  int64     `json:"clickhouse_writes"`
	FailedEvents      int64     `json:"failed_events"`
	LastError         string    `json:"last_error,omitempty"`
	CurrentSourceID   string    `json:"current_source_id"`
	CurrentInfoBaseID string    `json:"current_infobase_id"`
}

type Store struct {
	mu    sync.RWMutex
	state State
}

func New(serviceName, sourceID, infobaseID string) *Store {
	return &Store{state: State{
		ServiceName:       serviceName,
		Status:            "starting",
		StartedAt:         time.Now().UTC(),
		CurrentSourceID:   sourceID,
		CurrentInfoBaseID: infobaseID,
	}}
}

func (s *Store) SetStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Status = status
}

func (s *Store) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.state.LastError = ""
		return
	}
	s.state.LastError = err.Error()
	s.state.Status = "degraded"
}

func (s *Store) IncRead() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ReadEvents++
	s.state.LastEventAt = time.Now().UTC()
}

func (s *Store) IncBuffered() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.BufferedEvents++
}

func (s *Store) IncParsed(n int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ParsedEvents += n
}

func (s *Store) IncWritten(n int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.ClickHouseWrites += n
}

func (s *Store) IncFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.FailedEvents++
}

func (s *Store) Snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Store) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		state := s.Snapshot()
		if state.Status == "failed" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("failed"))
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.Snapshot())
	})
	return mux
}
