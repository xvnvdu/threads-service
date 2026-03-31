package repository

import (
	"context"

	"github.com/xvnvdu/threads-service/internal/domain"
)

type CommentSortOrder string

const (
	CommentSortNewestFirst CommentSortOrder = "NEWEST_FIRST"
	CommentSortOldestFirst CommentSortOrder = "OLDEST_FIRST"
)

// Repository описывает поведение методов для
// операций с основными сущностями сервиса
type Repository interface {
	// Операции с постами
	CreatePost(ctx context.Context, post *domain.Post) error
	GetPostByID(ctx context.Context, id string) (*domain.Post, error)
	GetPosts(ctx context.Context, limit, offset int) ([]*domain.Post, error)
	SetCommentsEnabled(ctx context.Context, postID string, enabled bool) (*domain.Post, error)
	DeletePost(ctx context.Context, id string) error

	// Операции с комментариями
	CreateComment(ctx context.Context, comment *domain.Comment) error
	GetCommentByID(ctx context.Context, id string) (*domain.Comment, error)
	GetComments(ctx context.Context, postID string, parentID *string, limit, offset int, order CommentSortOrder) ([]*domain.Comment, error)
	DeleteComment(ctx context.Context, id string) error
}
