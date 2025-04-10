package db

import (
	"fmt"
	"sync"
)

// Session представляет данные сессии пользователя.
type Session struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
	Token  string `json:"token"`
}

// SessionDB — простое in-memory хранилище сессий.
type SessionDB struct {
	Data map[string]*Session
	mu   sync.Mutex
}

// NewSessionDB создаёт новое хранилище сессий.
func NewSessionDB() *SessionDB {
	return &SessionDB{
		Data: make(map[string]*Session),
	}
}

// CreateSession генерирует и сохраняет сессию для данного userID.
func (sdb *SessionDB) CreateSession(userID string) (*Session, error) {
	sdb.mu.Lock()
	defer sdb.mu.Unlock()

	token := fmt.Sprintf("token_for_user_%s", userID)
	sess := &Session{
		ID:     token,
		UserID: userID,
		Token:  token,
	}
	sdb.Data[token] = sess
	return sess, nil
}

// DeleteSession удаляет сессию по токену.
func (sdb *SessionDB) DeleteSession(token string) error {
	sdb.mu.Lock()
	defer sdb.mu.Unlock()
	if _, ok := sdb.Data[token]; !ok {
		return fmt.Errorf("session not found")
	}
	delete(sdb.Data, token)
	return nil
}

// GetSessionByToken возвращает сессию по токену.
func (sdb *SessionDB) GetSessionByToken(token string) (*Session, error) {
	sdb.mu.Lock()
	defer sdb.mu.Unlock()
	sess, ok := sdb.Data[token]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return sess, nil
}
