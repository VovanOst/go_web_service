package repository

import (
	"errors"
	"rwa/internal/db"
)

type UserRepo struct {
	db *db.UserDB
}

func NewUserRepo(db *db.UserDB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(user *db.User) *db.User {
	return r.db.CreateUser(user)
}

// FindByEmail ищет пользователя по email.
// Перебираем всех пользователей, хранящихся в базе, и возвращаем того, чей email совпадает.
func (r *UserRepo) FindByEmail(email string) (*db.User, error) {
	for _, user := range r.db.Data { // убедитесь, что поле Data экспортировано (с большой буквы)
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

// FindByID ищет пользователя по его идентификатору.
func (r *UserRepo) FindByID(id string) (*db.User, error) {
	user, ok := r.db.Data[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}
