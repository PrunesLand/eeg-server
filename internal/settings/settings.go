package settings

import "sync"

// Settings holds the dynamic configuration of the server.
type Settings struct {
	mu   sync.RWMutex
	gain float64
}

// New creates a new Settings instance with defaults.
func New() *Settings {
	return &Settings{
		gain: 4.0, // Default Gain
	}
}

// GetGain safely returns the current gain.
func (s *Settings) GetGain() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gain
}

// SetGain safely updates the gain.
func (s *Settings) SetGain(g float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gain = g
}
