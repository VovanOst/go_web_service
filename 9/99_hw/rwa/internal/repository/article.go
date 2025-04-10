package repository

import "rwa/internal/db"

type ArticleRepo struct {
	db *db.ArticleDB
}

func NewArticleRepo(db *db.ArticleDB) *ArticleRepo {
	return &ArticleRepo{db: db}
}

func (r *ArticleRepo) Create(article *db.Article) *db.Article {
	return r.db.CreateArticle(article)
}

func (r *ArticleRepo) GetAll() []*db.Article {
	return r.db.GetAllArticles()
}

func (r *ArticleRepo) GetBySlug(slug string) (*db.Article, bool) {
	return r.db.GetArticleBySlug(slug)
}

func (r *ArticleRepo) Update(slug string, updated *db.Article) (*db.Article, bool) {
	return r.db.UpdateArticle(slug, updated)
}

func (r *ArticleRepo) Delete(slug string) bool {
	return r.db.DeleteArticle(slug)
}
