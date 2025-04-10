package repository

import (
	"errors"
	"rwa/internal/db"
)

type SessionRepo struct {
	db *db.SessionDB
}

func NewSessionRepo(db *db.SessionDB) *SessionRepo {
	return &SessionRepo{db: db}
}

// Create генерирует и сохраняет новую сессию для пользователя с заданным userID.
func (r *SessionRepo) Create(userID string) (*db.Session, error) {
	return r.db.CreateSession(userID)
}

// GetByToken ищет сессию по её токену.
func (r *SessionRepo) GetByToken(token string) (*db.Session, error) {
	sess, ok := r.db.Data[token]
	if !ok {
		return nil, errors.New("session not found")
	}
	return sess, nil
}

// Delete удаляет сессию по токену.
func (r *SessionRepo) Delete(token string) error {
	return r.db.DeleteSession(token)
}
