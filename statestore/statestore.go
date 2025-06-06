package statestore

import (
	"sync"
)

type StateInfo struct {
	ClientID       string
	RedirectURI    string
	RequestedScope string
}

// StateStore manages OAuth2 state values
type StateStore struct {
	states map[string]StateInfo
	mu     sync.RWMutex
}

// NewStateStore creates a new StateStore instance
func NewStateStore() *StateStore {
	return &StateStore{
		states: make(map[string]StateInfo),
	}
}

// Add stores a new state with client info
func (s *StateStore) Add(state, clientID, redirectURI, requestedScope string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state] = StateInfo{
		ClientID:       clientID,
		RedirectURI:    redirectURI,
		RequestedScope: requestedScope,
	}
}

func (s *StateStore) GetRequestedScope(state string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[state].RequestedScope
}

// ValidateState checks if a state exists without checking client info
func (s *StateStore) ValidateState(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.states[state]
	return exists
}

// ValidateWithClientInfo checks if a state exists and matches both clientID and redirectURI
func (s *StateStore) ValidateWithClientInfo(state, clientID, redirectURI string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, exists := s.states[state]
	if !exists {
		return false
	}
	if info.ClientID != clientID || info.RedirectURI != redirectURI {
		return false
	}
	return true
}

// ValidateRedirectURI checks if state exists and matches redirectURI without deleting
func (s *StateStore) ValidateRedirectURI(state, redirectURI string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, exists := s.states[state]
	if !exists {
		return false
	}

	return info.RedirectURI == redirectURI
}

// DeleteState removes the state from the store
func (s *StateStore) DeleteState(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, state)
}
