package main

import (
	"net/http"
	"rwa/internal/db"
	"rwa/internal/handler"
	"rwa/internal/repository"
)

func GetApp() http.Handler {
	mux := http.NewServeMux()

	// Инициализация in-memory "базы"
	userDB := db.NewUserDB()
	sessionDB := db.NewSessionDB()
	articleDB := db.NewArticleDB()

	// Репозитории
	userRepo := repository.NewUserRepo(userDB)
	sessionRepo := repository.NewSessionRepo(sessionDB)
	articleRepo := repository.NewArticleRepo(articleDB)

	// Хендлеры
	userHandler := handler.NewUserHandler(userRepo, sessionRepo)
	articleHandler := handler.NewArticleHandler(articleRepo, sessionRepo, userRepo)

	// Роуты
	mux.HandleFunc("/api/users", userHandler.UsersHandler)       // POST /api/users, GET /api/users
	mux.HandleFunc("/api/users/login", userHandler.LoginHandler) // POST login
	mux.HandleFunc("/api/user", userHandler.CurrentHandler)      // GET/PUT текущий юзер

	mux.HandleFunc("/api/articles", articleHandler.ListHandler)     // GET articles
	mux.HandleFunc("/api/articles/", articleHandler.ArticleHandler) // GET/PUT/DELETE article by slug

	mux.HandleFunc("/api/user/logout", userHandler.LogoutHandler)
	return mux
}
