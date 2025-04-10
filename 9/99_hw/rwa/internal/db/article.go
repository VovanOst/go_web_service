package db

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Article struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Body           string    `json:"body"`
	TagList        []string  `json:"tagList"`
	Slug           string    `json:"slug"`
	AuthorID       string    `json:"authorId"`
	Favorited      bool      `json:"favorited"`
	FavoritesCount int       `json:"favoritesCount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ArticleDB struct {
	data   map[string]*Article
	nextID int
	mu     sync.Mutex
}

func NewArticleDB() *ArticleDB {
	return &ArticleDB{
		data:   make(map[string]*Article),
		nextID: 1,
	}
}

func (adb *ArticleDB) CreateArticle(article *Article) *Article {
	adb.mu.Lock()
	defer adb.mu.Unlock()

	article.ID = adb.nextID
	adb.nextID++

	article.Slug = fmt.Sprintf("%s-%d", slugify(article.Title), article.ID)
	article.CreatedAt = time.Now().UTC()
	article.UpdatedAt = article.CreatedAt

	adb.data[article.Slug] = article
	return article
}

func (adb *ArticleDB) GetAllArticles() []*Article {
	adb.mu.Lock()
	defer adb.mu.Unlock()

	articles := make([]*Article, 0, len(adb.data))
	for _, art := range adb.data {
		articles = append(articles, art)
	}
	return articles
}

func (adb *ArticleDB) GetArticleBySlug(slug string) (*Article, bool) {
	adb.mu.Lock()
	defer adb.mu.Unlock()

	art, ok := adb.data[slug]
	return art, ok
}

func (adb *ArticleDB) UpdateArticle(slug string, updated *Article) (*Article, bool) {
	adb.mu.Lock()
	defer adb.mu.Unlock()

	art, ok := adb.data[slug]
	if !ok {
		return nil, false
	}

	if updated.Title != "" {
		art.Title = updated.Title
	}
	if updated.Description != "" {
		art.Description = updated.Description
	}
	if updated.Body != "" {
		art.Body = updated.Body
	}
	if updated.TagList != nil {
		art.TagList = updated.TagList
	}

	art.UpdatedAt = time.Now().UTC()
	return art, true
}

func (adb *ArticleDB) DeleteArticle(slug string) bool {
	adb.mu.Lock()
	defer adb.mu.Unlock()

	if _, ok := adb.data[slug]; ok {
		delete(adb.data, slug)
		return true
	}
	return false
}

func slugify(title string) string {
	return strings.ToLower(strings.ReplaceAll(title, " ", "-"))
}
