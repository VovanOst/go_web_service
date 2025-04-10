package db

import (
	"strconv"
	"sync"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`   // не возвращается в JSON
	Bio       string    `json:"bio"` // поле для описания
	Image     string    `json:"image"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserDB struct {
	Data   map[string]*User
	nextID int
	mu     sync.Mutex
}

func NewUserDB() *UserDB {
	return &UserDB{
		Data:   make(map[string]*User),
		nextID: 1,
	}
}

// CreateUser сохраняет нового пользователя в in-memory базе.
func (udb *UserDB) CreateUser(user *User) *User {
	udb.mu.Lock()
	defer udb.mu.Unlock()

	// Генерируем идентификатор как строку из nextID
	user.ID = strconv.Itoa(udb.nextID)
	udb.nextID++
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = user.CreatedAt

	udb.Data[user.ID] = user
	return user
}
