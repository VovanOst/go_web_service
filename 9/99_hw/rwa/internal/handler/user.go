package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"rwa/internal/db"
	"rwa/internal/repository"
)

type UserHandler struct {
	userRepo    *repository.UserRepo
	sessionRepo *repository.SessionRepo
}

func NewUserHandler(u *repository.UserRepo, s *repository.SessionRepo) *UserHandler {
	return &UserHandler{userRepo: u, sessionRepo: s}
}

// UsersHandler обрабатывает запросы, связанные с пользователями.
// Для метода POST осуществляется регистрация.
func (h *UserHandler) UsersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.registerUser(w, r)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func (h *UserHandler) registerUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		User struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Username string `json:"username"`
		} `json:"user"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Создание нового пользователя
	newUser := &db.User{
		Username:  payload.User.Username,
		Email:     payload.User.Email,
		Password:  payload.User.Password,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Bio:       "",
		Image:     "",
	}

	created := h.userRepo.Create(newUser)

	// Создаем сессию для вновь зарегистрированного пользователя,
	// чтобы вернуть токен в ответе
	session, err := h.sessionRepo.Create(created.ID)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"user": map[string]interface{}{
			"email":     created.Email,
			"username":  created.Username,
			"bio":       created.Bio,
			"image":     created.Image,
			"token":     session.Token,
			"createdAt": created.CreatedAt.Format(time.RFC3339),
			"updatedAt": created.UpdatedAt.Format(time.RFC3339),
		},
	}

	writeJSON(w, response, http.StatusCreated)
}

// LoginHandler обрабатывает запрос логина пользователя.
func (h *UserHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Принимаем только POST-запросы
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Декодирование JSON-пейлоада: ожидаем {"user": {"email": "...", "password": "..."}}
	var payload struct {
		User struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Поиск пользователя по email через репозиторий
	user, err := h.userRepo.FindByEmail(payload.User.Email)
	if err != nil || user == nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	// Простая проверка пароля (в реальном приложении следует использовать хеширование)
	if user.Password != payload.User.Password {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Создаем сессию через SessionRepo, которая возвращает объект сессии с полем Token
	session, err := h.sessionRepo.Create(user.ID)
	log.Println("LOGIN USER:", user.ID, "TOKEN:", session.Token)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	// Формирование ответа. В тестах ожидается, что поля createdAt и updatedAt представлены в формате времени.
	response := map[string]interface{}{
		"user": map[string]interface{}{
			"email":     user.Email,
			"username":  user.Username,
			"bio":       user.Bio,
			"image":     user.Image,
			"token":     session.Token,
			"createdAt": user.CreatedAt.Format(time.RFC3339),
			"updatedAt": user.UpdatedAt.Format(time.RFC3339),
		},
	}

	// Функция writeJSON должна устанавливать заголовки и записывать JSON в ответ
	writeJSON(w, response, http.StatusOK)
}

func (h *UserHandler) CurrentHandler(w http.ResponseWriter, r *http.Request) {
	// Извлекаем заголовок Authorization; ожидается формат "Token <value>"
	authHeader := r.Header.Get("Authorization")
	const tokenPrefix = "Token "
	if authHeader == "" || !strings.HasPrefix(authHeader, tokenPrefix) {
		http.Error(w, "authorization token missing or invalid", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, tokenPrefix)

	// Получаем сессию по токену
	session, err := h.sessionRepo.GetByToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Находим пользователя по идентификатору, сохранённому в сессии
	user, err := h.userRepo.FindByID(session.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	// Обработка GET и PUT-запросов по одному эндпоинту /api/user
	switch r.Method {
	case http.MethodGet:
		// GET возвращает данные, где поле token – пустое (как требуется тестами)
		response := map[string]interface{}{
			"user": map[string]interface{}{
				"email":     user.Email,
				"username":  user.Username,
				"token":     "",
				"bio":       user.Bio,
				"image":     user.Image,
				"createdAt": user.CreatedAt.Format(time.RFC3339),
				"updatedAt": user.UpdatedAt.Format(time.RFC3339),
			},
		}
		writeJSON(w, response, http.StatusOK)
	case http.MethodPut:
		// PUT ожидает JSON вида: {"user": {"email": "...", "bio": "..." }}
		var payload struct {
			User struct {
				Email string `json:"email"`
				Bio   string `json:"bio"`
			} `json:"user"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		// Обновляем поля пользователя, если они непустые
		if payload.User.Email != "" {
			user.Email = payload.User.Email
		}
		if payload.User.Bio != "" {
			user.Bio = payload.User.Bio
		}
		user.UpdatedAt = time.Now().UTC()

		// Для обеспечения корректной авторизации последующих запросов возвращаем в ответе
		// сессионный токен (чтобы тестовый After-функция обновила значение токена).
		response := map[string]interface{}{
			"user": map[string]interface{}{
				"email":     user.Email,
				"username":  user.Username,
				"token":     session.Token, // возвращаем исходный токен,
				"bio":       user.Bio,
				"image":     user.Image,
				"createdAt": user.CreatedAt.Format(time.RFC3339),
				"updatedAt": user.UpdatedAt.Format(time.RFC3339),
			},
		}
		writeJSON(w, response, http.StatusOK)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// LogoutHandler обрабатывает выход пользователя.
func (h *UserHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Извлекаем заголовок Authorization, ожидается "Token <value>"
	authHeader := r.Header.Get("Authorization")
	const tokenPrefix = "Token "
	if authHeader == "" || !strings.HasPrefix(authHeader, tokenPrefix) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, tokenPrefix)

	// Удаляем сессию по токену
	if err := h.sessionRepo.Delete(token); err != nil {
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
