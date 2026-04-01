package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/xvnvdu/threads-service/internal/domain"
	"github.com/xvnvdu/threads-service/internal/repository"
)

const (
	MaxPaginationLimit = 100
	MaxCommentLength   = 2000
)

type service struct {
	repo repository.Repository
}

type Service interface {
	CreatePost(ctx context.Context, post *domain.Post) (*domain.Post, error)
	GetPostByID(ctx context.Context, id string) (*domain.Post, error)
	GetPosts(ctx context.Context, limit, offset int) ([]*domain.Post, error)
	SetCommentsEnabled(ctx context.Context, postID string, enabled bool) (*domain.Post, error)
	DeletePost(ctx context.Context, id string) error

	CreateComment(ctx context.Context, comment *domain.Comment) (*domain.Comment, error)
	GetCommentByID(ctx context.Context, id string) (*domain.Comment, error)
	GetComments(ctx context.Context, postID string, parentID *string, limit, offset int, order repository.CommentSortOrder) ([]*domain.Comment, error)
	DeleteComment(ctx context.Context, id string) error
}

// Проверяем, что объект сервиса корректно реализует интерфейс
var _ Service = (*service)(nil)

func NewService(repo repository.Repository) Service {
	return &service{repo: repo}
}

// CreatePost валидирует данные поста и передает его на сохранение в бд
func (s *service) CreatePost(ctx context.Context, post *domain.Post) (*domain.Post, error) {
	if post.ID == "" {
		post.ID = uuid.New().String()
	}
	if post.CreatedAt.IsZero() {
		post.CreatedAt = time.Now()
	}

	if post.AuthorID == "" {
		return nil, fmt.Errorf("service: post cannot be created by author with empty id")
	}
	if post.Title == "" {
		return nil, fmt.Errorf("service: cannot create post without a title")
	}
	if post.Content == "" {
		return nil, fmt.Errorf("service: cannot create empty post")
	}

	log.Println("[INFO] service: validated post data:", post.ID)
	if err := s.repo.CreatePost(ctx, post); err != nil {
		return nil, fmt.Errorf("service: failed to create post: %w", err)
	}
	return post, nil
}

// GetPostByID обращается к бд для получения поста по его id
func (s *service) GetPostByID(ctx context.Context, id string) (*domain.Post, error) {
	if id == "" {
		return nil, fmt.Errorf("service: cannot get comment with empty id")
	}
	log.Println("[INFO] service: validated id to get post data:", id)

	post, err := s.repo.GetPostByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get post by ID: %w", err)
	}
	return post, nil
}

// GetPosts валидирует получаемые параметры и возвращает посты из бд
func (s *service) GetPosts(ctx context.Context, limit, offset int) ([]*domain.Post, error) {
	if limit <= 0 || offset < 0 {
		return nil, fmt.Errorf(
			"service: failed to get posts: limit must be positive and offset must be non-negative",
		)
	}
	if limit > MaxPaginationLimit {
		limit = MaxPaginationLimit
	}
	log.Println("[INFO] service: validated limit and offset to get post data")

	posts, err := s.repo.GetPosts(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get posts: %w", err)
	}
	return posts, nil
}

// SetCommentsEnabled обращается к бд и вкл/выкл комменты для указанного поста
func (s *service) SetCommentsEnabled(ctx context.Context, postID string, enabled bool) (*domain.Post, error) {
	if postID == "" {
		return nil, fmt.Errorf("service: cannot switch comments state: post id must not be empty")
	}
	post, err := s.repo.SetCommentsEnabled(ctx, postID, enabled)
	if err != nil {
		return nil, fmt.Errorf("service: failed to switch comments state for post with id %s: %w", postID, err)
	}
	return post, nil
}

// DeletePost обращается к бд для удаления поста по id
func (s *service) DeletePost(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("service: cannot delete post with empty id")
	}
	log.Println("[INFO] service: validated id to delete post:", id)

	if err := s.repo.DeletePost(ctx, id); err != nil {
		return fmt.Errorf("service: failed to delete post: %w", err)
	}
	return nil
}

// CreateComment валидирует корректность комментария и передает его на сохранение в бд
func (s *service) CreateComment(ctx context.Context, comment *domain.Comment) (*domain.Comment, error) {
	if comment.Content == "" {
		return nil, fmt.Errorf("service: failed to create comment: comment must not be empty")
	}
	if len(comment.Content) > MaxCommentLength {
		return nil, fmt.Errorf("service: failed to create comment: comment content exceeds maximum allowed length")
	}

	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now()
	}

	if comment.AuthorID == "" {
		return nil, fmt.Errorf("service: failed to create comment: comment must have its author with non-empty id")
	}
	if comment.PostID == "" {
		return nil, fmt.Errorf("service: failed to create comment: comment must be created under a post with non-empty id")
	}

	log.Println("[INFO] service: validated comment data:", comment.ID)
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, fmt.Errorf("service: failed to create comment: %w", err)
	}
	return comment, nil
}

// GetCommentByID обращается к бд и получает комментарий по указанному id
func (s *service) GetCommentByID(ctx context.Context, id string) (*domain.Comment, error) {
	if id == "" {
		return nil, fmt.Errorf("service: cannot get comment with empty id")
	}
	log.Println("[INFO] service: validated id to get post data:", id)

	comment, err := s.repo.GetCommentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get comment by id: %w", err)
	}
	return comment, nil
}

// GetComments валидирует передаваемые параметры и возвращает комментарии из бд
func (s *service) GetComments(ctx context.Context, postID string, parentID *string, limit, offset int, order repository.CommentSortOrder) ([]*domain.Comment, error) {
	if limit <= 0 || offset < 0 {
		return nil, fmt.Errorf(
			"service: failed to get comments: limit must be positive and offset must be non-negative",
		)
	}
	if limit > MaxPaginationLimit {
		limit = MaxPaginationLimit
	}

	// Если хотим получить не комментарии верхнего уровня,
	// а ответы на них - ответы сортируются по хронологии
	if parentID != nil {
		order = repository.CommentSortOldestFirst
	}
	log.Println("[INFO] service: validated data to get comments")

	flatComments, err := s.repo.GetComments(ctx, postID, parentID, limit, offset, order)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get comments: %w", err)
	}

	return flatComments, nil
}

// DeleteComment обращается к бд для удаления комментария по id
func (s *service) DeleteComment(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("service: cannot delete comment with empty id")
	}
	log.Println("[INFO] service: validated id to delete post:", id)

	if err := s.repo.DeleteComment(ctx, id); err != nil {
		return fmt.Errorf("service: failed to delete comment: %w", err)
	}
	return nil
}
