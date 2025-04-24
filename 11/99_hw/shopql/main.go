package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"hw11_shopql/graph"
	"hw11_shopql/graph/model"
	"hw11_shopql/middleware"
	"hw11_shopql/service"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GetApp возвращает HTTP handler для тестов и сервера
func GetApp() http.Handler {
	svc := service.NewService()
	resolver := &graph.Resolver{Svc: svc}
	srv := handler.NewDefaultServer(
		graph.NewExecutableSchema(graph.Config{Resolvers: resolver}),
	)

	mux := http.NewServeMux()
	// GraphQL playground
	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))
	// GraphQL endpoint с middleware авторизации
	mux.Handle("/query", middleware.AuthMiddleware(srv))
	// Регистрация пользователя
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			User struct {
				Email    string `json:"email"`
				Password string `json:"password"`
				Username string `json:"username"`
			} `json:"user"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		token := fmt.Sprintf("%d", rand.Int63())
		// Инициализируем пустую корзину для нового пользователя
		service.NewService().Carts[token] = make(map[string]*model.CartItem)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"body": map[string]string{"token": token}})
	})

	return mux
}

// dd — утилита для тестов: выводит expected и got
func dd(a, b interface{}) {
	fmt.Printf("Expected: %#v\nGot: %#v\n", a, b)
}

func main() {
	app := GetApp()
	log.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", app))
}
