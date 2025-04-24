package graph

import (
	"context"
	"errors"
	"hw11_shopql/middleware"
)

// GetUserIDFromContext извлекает идентификатор пользователя из контекста.
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(middleware.UserIDCtxKey).(string)
	if !ok || userID == "" {
		return "", errors.New("User not authorized")
	}
	return userID, nil
}
