package graph

import (
	"context"
	"errors"
	"github.com/99designs/gqlgen/graphql"
)

// directiveResolver реализует методы для директив.
type directiveResolver struct{ *Resolver }

// Authorized – реализация директивы @authorized.
func (r *directiveResolver) Authorized(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
	// Проверяем наличие идентификатора пользователя в контексте.
	if _, err := GetUserIDFromContext(ctx); err != nil {
		return nil, errors.New("User not authorized")
	}
	return next(ctx)
}
