package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type SessionData struct {
	Username  string
	ExpiresAt time.Time
}

type SessionManager struct {
	sessions sync.Map
	duration time.Duration
}

func NewSessionManager(duration time.Duration) *SessionManager {
	if duration <= 0 {
		duration = 1 * time.Hour
	}
	sm := &SessionManager{
		duration: duration,
	}
	// Start periodic cleanup goroutine for expired sessions
	go sm.startCleanupLoop()
	return sm
}

func (sm *SessionManager) CreateSession(username string) (string, time.Time) {
	token := generateRandomToken()
	expiresAt := time.Now().Add(sm.duration)

	sm.sessions.Store(token, SessionData{
		Username:  username,
		ExpiresAt: expiresAt,
	})

	return token, expiresAt
}

func (sm *SessionManager) ValidateSession(token string) bool {
	if token == "" {
		return false
	}

	val, ok := sm.sessions.Load(token)
	if !ok {
		return false
	}

	data, ok := val.(SessionData)
	if !ok {
		return false
	}

	if time.Now().After(data.ExpiresAt) {
		sm.sessions.Delete(token)
		return false
	}

	return true
}

func (sm *SessionManager) DeleteSession(token string) {
	if token != "" {
		sm.sessions.Delete(token)
	}
}

func (sm *SessionManager) startCleanupLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	for range ticker.C {
		now := time.Now()
		sm.sessions.Range(func(key, value interface{}) bool {
			if data, ok := value.(SessionData); ok {
				if now.After(data.ExpiresAt) {
					sm.sessions.Delete(key)
				}
			}
			return true
		})
	}
}

func generateRandomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
