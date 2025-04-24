package middleware

import (
	"context"
	"net/http"
	"strings"
)

// Объявляем тип ключа для контекста, чтобы избежать конфликтов.
type contextKey string

// Определяем экспортируемый ключ для идентификатора пользователя.
const UserIDCtxKey contextKey = "userID"

// AuthMiddleware извлекает токен из заголовка Authorization и кладёт userID в контекст.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем наличие заголовка Authorization вида "Token <userID>"
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Token ") {
			userID := strings.TrimPrefix(authHeader, "Token ")
			// Помещаем идентификатор в контекст.
			ctx := context.WithValue(r.Context(), UserIDCtxKey, userID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
