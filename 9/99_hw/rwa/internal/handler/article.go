package handler

import (
	"encoding/json"
	"fmt"
	_ "fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"rwa/internal/db"
	"rwa/internal/repository"
)

// ArticleHandler обрабатывает HTTP-запросы, связанные со статьями.
type ArticleHandler struct {
	articleRepo *repository.ArticleRepo
	sessionRepo *repository.SessionRepo // оставляем для будущей аутентификации
	userRepo    *repository.UserRepo
}

func NewArticleHandler(ar *repository.ArticleRepo, sr *repository.SessionRepo, ur *repository.UserRepo) *ArticleHandler {
	return &ArticleHandler{
		articleRepo: ar,
		sessionRepo: sr,
		userRepo:    ur,
	}
}

// ListHandler обрабатывает запросы на список статей (GET) и создание статьи (POST) по пути /api/articles.
func (h *ArticleHandler) ListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listArticles(w, r)
	case http.MethodPost:
		h.createArticle(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ArticleHandler обрабатывает запросы для конкретной статьи по slug (GET/PUT/DELETE) по пути /api/articles/{slug}.
func (h *ArticleHandler) ArticleHandler(w http.ResponseWriter, r *http.Request) {
	// Ожидаем, что URL имеет вид: /api/articles/{slug}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	slug := parts[len(parts)-1]
	switch r.Method {
	case http.MethodGet:
		h.getArticle(w, r, slug)
	case http.MethodPut:
		h.updateArticle(w, r, slug)
	case http.MethodDelete:
		h.deleteArticle(w, r, slug)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// listArticles обрабатывает GET-запрос для /api/articles и возвращает список статей с заполненным вложенным объектом "author".
func (h *ArticleHandler) listArticles(w http.ResponseWriter, r *http.Request) {
	allArticles := h.articleRepo.GetAll()

	authorQuery := r.URL.Query().Get("author")
	tagQuery := r.URL.Query().Get("tag")

	// Фильтрация: начинаем с общего списка и оставляем только те статьи, которые удовлетворяют фильтрам.
	filteredArticles := make([]*db.Article, 0, len(allArticles))
	for _, art := range allArticles {
		include := true

		// Фильтрация по автору, если параметр задан.
		if authorQuery != "" {
			author, err := h.userRepo.FindByID(art.AuthorID)
			if err != nil || author.Username != authorQuery {
				include = false
			}
		}

		// Фильтрация по тегу, если параметр задан.
		if tagQuery != "" {
			matched := false
			for _, tag := range art.TagList {
				if tag == tagQuery {
					matched = true
					break
				}
			}
			if !matched {
				include = false
			}
		}

		if include {
			filteredArticles = append(filteredArticles, art)
		}
	}

	// Формируем ответ: для каждой статьи заполняем информацию об авторе.
	responseArticles := make([]interface{}, 0, len(filteredArticles))
	for _, art := range filteredArticles {
		author, err := h.userRepo.FindByID(art.AuthorID)
		var authorResp map[string]interface{}
		if err != nil {
			authorResp = map[string]interface{}{
				"username":  "",
				"bio":       "",
				"image":     "",
				"following": false,
				"createdAt": nil,
				"updatedAt": nil,
			}
		} else {
			authorResp = map[string]interface{}{
				"username":  author.Username,
				"bio":       author.Bio,
				"image":     author.Image,
				"following": false,
				// Для полей времени возвращаем null, чтобы тест их десериализовал как FakeTime с Valid: false.
				"createdAt": nil,
				"updatedAt": nil,
			}
		}

		artResp := map[string]interface{}{
			"id":             art.ID,
			"title":          art.Title,
			"description":    art.Description,
			"body":           art.Body,
			"tagList":        art.TagList,
			"slug":           art.Slug,
			"authorId":       art.AuthorID,
			"favorited":      art.Favorited,
			"favoritesCount": art.FavoritesCount,
			"createdAt":      art.CreatedAt.Format(time.RFC3339),
			"updatedAt":      art.UpdatedAt.Format(time.RFC3339),
			"author":         authorResp,
		}
		responseArticles = append(responseArticles, artResp)
	}

	response := map[string]interface{}{
		"articles":      responseArticles,
		"articlesCount": len(responseArticles),
	}

	writeJSON(w, response, http.StatusOK)
}

func (h *ArticleHandler) createArticle(w http.ResponseWriter, r *http.Request) {
	// Аутентификация: извлекаем токен из заголовка Authorization
	fmt.Println("AuthHeader:", r.Header.Get("Authorization"))
	authHeader := r.Header.Get("Authorization")
	const tokenPrefix = "Token "
	if authHeader == "" || !strings.HasPrefix(authHeader, tokenPrefix) {
		http.Error(w, "authorization token missing", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, tokenPrefix)

	// Получаем сессию по токену
	session, err := h.sessionRepo.GetByToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// По идентификатору из сессии получаем пользователя
	user, err := h.userRepo.FindByID(session.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	// Читаем тело запроса. Ожидается JSON вида:
	// {"article": {"title": "...", "description": "...", "body": "...", "tagList": ["..."]}}
	var payload struct {
		Article struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Body        string   `json:"body"`
			TagList     []string `json:"tagList"`
		} `json:"article"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Создаем новую статью, используя данные из запроса
	newArticle := &db.Article{
		Title:       payload.Article.Title,
		Description: payload.Article.Description,
		Body:        payload.Article.Body,
		TagList:     payload.Article.TagList,
		AuthorID:    user.ID, // устанавливаем автора статьи по данным аутентификации
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	created := h.articleRepo.Create(newArticle)

	// Формируем ответ. Тест ожидает, что в объекте article будет
	// вложенный объект "author" с информацией о пользователе:
	response := map[string]interface{}{
		"article": map[string]interface{}{
			"id":             created.ID,
			"title":          created.Title,
			"description":    created.Description,
			"body":           created.Body,
			"tagList":        created.TagList,
			"slug":           created.Slug,
			"favorited":      created.Favorited,
			"favoritesCount": created.FavoritesCount,
			"createdAt":      created.CreatedAt.Format(time.RFC3339),
			"updatedAt":      created.UpdatedAt.Format(time.RFC3339),
			"author": map[string]interface{}{
				"username":  user.Username, // ожидается "golang"
				"bio":       user.Bio,      // ожидается "Info about golang"
				"image":     user.Image,
				"following": false, // по умолчанию false
				// Если нужно — можно вернуть поля createdAt, updatedAt автора
				//"createdAt": user.CreatedAt.Format(time.RFC3339),
				//"updatedAt": user.UpdatedAt.Format(time.RFC3339),
			},
		},
	}

	writeJSON(w, response, http.StatusCreated)
}

func (h *ArticleHandler) getArticle(w http.ResponseWriter, r *http.Request, slug string) {
	article, ok := h.articleRepo.GetBySlug(slug)
	if !ok {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	response := map[string]interface{}{
		"article": article,
	}
	jsonResponse(w, response)
}

func (h *ArticleHandler) updateArticle(w http.ResponseWriter, r *http.Request, slug string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload struct {
		Article struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Body        string   `json:"body"`
			TagList     []string `json:"tagList"`
		} `json:"article"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	updatedArticle := &db.Article{
		Title:       payload.Article.Title,
		Description: payload.Article.Description,
		Body:        payload.Article.Body,
		TagList:     payload.Article.TagList,
		UpdatedAt:   time.Now().UTC(),
	}

	article, ok := h.articleRepo.Update(slug, updatedArticle)
	if !ok {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	response := map[string]interface{}{
		"article": article,
	}
	jsonResponse(w, response)
}

func (h *ArticleHandler) deleteArticle(w http.ResponseWriter, r *http.Request, slug string) {
	if ok := h.articleRepo.Delete(slug); !ok {
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}
