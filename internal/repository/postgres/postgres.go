package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/xvnvdu/threads-service/internal/domain"
	repo "github.com/xvnvdu/threads-service/internal/repository"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dataSourceName string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &PostgresRepository{db: db}, nil
}

// Проверяем на удовлетворение интерфейсу
var _ repo.Repository = (*PostgresRepository)(nil)

// CreatePost принимает пост и сохраняет его в бд
func (r *PostgresRepository) CreatePost(ctx context.Context, post *domain.Post) error {
	query := `
		INSERT INTO posts (id, title, content, author_id, comments_enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		post.ID,
		post.Title,
		post.Content,
		post.AuthorID,
		post.CommentsEnabled,
		post.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	log.Println("[INFO] successfully created and saved post to postgres storage:", post.ID)
	return nil
}

// GetPostByID извлекает пост из бд по id, если таковой имеется
func (r *PostgresRepository) GetPostByID(ctx context.Context, id string) (*domain.Post, error) {
	query := `
		SELECT id, title, content, author_id, comments_enabled, created_at
		FROM posts
		WHERE id = $1
	`
	var post domain.Post

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID,
		&post.Title,
		&post.Content,
		&post.AuthorID,
		&post.CommentsEnabled,
		&post.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("[WARN] post not found in postgres storage:", id)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get post by id: %w", err)
	}

	log.Println("[INFO] successfully retrieved post from postgres storage:", post.ID)
	return &post, nil
}

// GetPosts возвращает из бд limit постов с указанным offset
func (r *PostgresRepository) GetPosts(ctx context.Context, limit, offset int) ([]*domain.Post, error) {
	query := `
		SELECT id, title, content, author_id, comments_enabled, created_at
		FROM posts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var posts []*domain.Post
	for rows.Next() {
		var p domain.Post
		if err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Content,
			&p.AuthorID,
			&p.CommentsEnabled,
			&p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post row: %w", err)
		}
		posts = append(posts, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	log.Println("[INFO] successfully retrieved posts from postgres storage")
	return posts, nil
}

// SetCommentsEnabled переключает возможность оставлять комментарии под постом
func (r *PostgresRepository) SetCommentsEnabled(ctx context.Context, postID string, enabled bool) (*domain.Post, error) {
	query := `
		UPDATE posts
		SET comments_enabled = $1
		WHERE id = $2
		RETURNING id, title, content, author_id, comments_enabled, created_at
	`
	var p domain.Post
	err := r.db.QueryRowContext(ctx, query, enabled, postID).Scan(
		&p.ID,
		&p.Title,
		&p.Content,
		&p.AuthorID,
		&p.CommentsEnabled,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("post not found: %s", postID)
		}
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	log.Println("[INFO] successfully set CommentsEnabled:", enabled)
	return &p, nil
}

// DeletePost удаляет пост и все связанные с ним комментарии
func (r *PostgresRepository) DeletePost(ctx context.Context, id string) error {
	query := `DELETE FROM posts WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("post with ID %s not found", id)
	}

	log.Println("[INFO] successfully deleted post from postgres storage:", id)
	return nil
}

// CreateComment сохраняет комментарий в бд
func (r *PostgresRepository) CreateComment(ctx context.Context, comment *domain.Comment) error {
	query := `
		INSERT INTO comments (id, post_id, parent_id, content, author_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	var parentID sql.NullString
	if comment.ParentID != nil {
		parentID = sql.NullString{String: *comment.ParentID, Valid: true}
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		comment.ID,
		comment.PostID,
		parentID,
		comment.Content,
		comment.AuthorID,
		comment.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	log.Println("[INFO] successfully created and saved comment to postgres storage:", comment.ID)
	return nil
}

// GetCommentByID возвращает комментарий из бд, если таковой имеется
func (r *PostgresRepository) GetCommentByID(ctx context.Context, id string) (*domain.Comment, error) {
	query := `
		SELECT id, post_id, parent_id, content, author_id, created_at
		FROM comments
		WHERE id = $1
	`
	var c domain.Comment
	var parentID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.PostID,
		&parentID,
		&c.Content,
		&c.AuthorID,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("[WARN] could not find comment in postgres storage:", id)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}
	if parentID.Valid {
		c.ParentID = &parentID.String
	} else {
		c.ParentID = nil
	}

	log.Println("[INFO] successfully retrieved comment from postgres storage:", id)
	return &c, nil
}

// GetComments возвращает limit комментариев под постом с указанными offset и родителем
func (r *PostgresRepository) GetComments(ctx context.Context, postID string, parentID *string, limit, offset int, order repo.CommentSortOrder) ([]*domain.Comment, error) {
	comments := []*domain.Comment{}

	query := `
		SELECT id, post_id, parent_id, content, author_id, created_at
		FROM comments
		WHERE post_id = $1
	`
	args := []any{postID}

	// Проверяем root/не root
	if parentID != nil {
		query += " AND parent_id = $2"
		args = append(args, *parentID)
	} else {
		query += " AND parent_id IS NULL"
	}

	// Сортировка (либо root комменты, либо ответы на коммент)
	if order == repo.CommentSortNewestFirst {
		query += " ORDER BY created_at DESC"
	} else {
		query += " ORDER BY created_at ASC"
	}

	// Если был root, то на данном этапе аргумент пока один
	// А если не root, но аргументов на один больше
	if parentID != nil {
		query += " LIMIT $3 OFFSET $4"
		args = append(args, limit, offset)
	} else {
		query += " LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c domain.Comment
		var parent sql.NullString
		if err := rows.Scan(&c.ID, &c.PostID, &parent, &c.Content, &c.AuthorID, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		if parent.Valid {
			c.ParentID = &parent.String
		}
		comments = append(comments, &c)
	}

	log.Println("[INFO] successfully retrieved comments from postgres storage")
	return comments, nil
}

// DeleteComment удаляет комментарий и все его дочерние комментарии из бд
func (r *PostgresRepository) DeleteComment(ctx context.Context, id string) error {
	query := `DELETE FROM comments WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("comment not found")
	}

	log.Println("[INFO] successfully deleted comment from postgres storage:", id)
	return nil
}

// Close закрывает соединение с бд
func (r *PostgresRepository) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}
