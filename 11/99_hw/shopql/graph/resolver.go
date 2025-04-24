package graph

import (
	"context"
	"errors"
	"github.com/99designs/gqlgen/graphql"
	"hw11_shopql/service"
	"log"
)

// Resolver — ваша корневая структура, сгенерированная gqlgen.
type Resolver struct {
	Svc *service.Service
}

// Directives возвращает все ваши директивы
func (r *Resolver) Directives() DirectiveRoot {
	return DirectiveRoot{
		// имя поля CamelCase от вашей директивы @authorized
		Authorized: func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
			log.Printf("[DEBUG] Directives %v", ctx)
			_, err := GetUserIDFromContext(ctx)
			if err != nil {
				return nil, errors.New("User not authorized")
			}
			return next(ctx)
		},
	}
}
