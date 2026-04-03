package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xvnvdu/threads-service/internal/domain"
	"github.com/xvnvdu/threads-service/internal/repository"
	"github.com/xvnvdu/threads-service/internal/repository/mocks"
	"go.uber.org/mock/gomock"
)

func TestServiceCreatePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	s := NewService(mockRepo, NewCommentPubSub())

	ctx := context.Background()

	tests := []struct {
		name        string
		post        *domain.Post
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name: "successful creation with empty id and createdAt",
			post: &domain.Post{
				AuthorID: "user1",
				Title:    "Test Title",
				Content:  "Test Content",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreatePost(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, post *domain.Post) error {
						t.Logf("mock createPost called with id: %s", post.ID)
						assert.NotEmpty(t, post.ID)
						assert.False(t, post.CreatedAt.IsZero())
						return nil
					})
			},
		},
		{
			name: "empty author id",
			post: &domain.Post{
				Title:   "Test Title",
				Content: "Test Content",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "author with empty id",
		},
		{
			name: "empty title",
			post: &domain.Post{
				AuthorID: "user1",
				Content:  "Test Content",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "without a title",
		},
		{
			name: "empty content",
			post: &domain.Post{
				AuthorID: "user1",
				Title:    "Test Title",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "empty post",
		},
		{
			name: "repository error",
			post: &domain.Post{
				AuthorID: "user1",
				Title:    "Test Title",
				Content:  "Test Content",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreatePost(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to create post",
		},
		{
			name: "id and createdAt already set",
			post: &domain.Post{
				ID:        uuid.New().String(),
				CreatedAt: time.Now(),
				AuthorID:  "user1",
				Title:     "Test Title",
				Content:   "Test Content",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreatePost(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			result, err := s.CreatePost(ctx, tt.post)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			t.Logf("post created: %s", result.ID)
		})
	}
}

func TestServiceGetPostByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	s := NewService(mockRepo, NewCommentPubSub())

	ctx := context.Background()

	tests := []struct {
		name        string
		id          string
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty id",
			id:          "",
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "service: cannot get post with empty id",
		},
		{
			name: "successful retrieval",
			id:   "post1",
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPostByID(gomock.Any(), "post1").
					Return(&domain.Post{ID: "post1"}, nil)
			},
		},
		{
			name: "repository error",
			id:   "post2",
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPostByID(gomock.Any(), "post2").
					Return(nil, fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to get post by ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			post, err := s.GetPostByID(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, post)

			t.Logf("post retrieved: %s", post.ID)
		})
	}
}

func TestServiceGetPosts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	s := NewService(mockRepo, NewCommentPubSub())

	ctx := context.Background()

	tests := []struct {
		name        string
		limit       int
		offset      int
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful retrieval",
			limit: 10, offset: 0,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPosts(gomock.Any(), 10, 0).
					Return([]*domain.Post{{ID: "p1"}}, nil)
			},
		},
		{
			name:        "invalid limit",
			limit:       0,
			offset:      0,
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "limit must be positive",
		},
		{
			name:        "invalid offset",
			limit:       5,
			offset:      -1,
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "offset must be non-negative",
		},
		{
			name:   "limit exceeds max pagination limit",
			limit:  MaxPaginationLimit + 100,
			offset: 0,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPosts(gomock.Any(), MaxPaginationLimit, 0).
					Return([]*domain.Post{{ID: "p1"}}, nil)
			},
			wantErr: false,
		},
		{
			name:  "repository error",
			limit: 5, offset: 0,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPosts(gomock.Any(), 5, 0).
					Return(nil, fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to get posts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			posts, err := s.GetPosts(ctx, tt.limit, tt.offset)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, posts)

			t.Logf("posts retrieved: %d", len(posts))
		})
	}
}

func TestServiceSetCommentsEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	s := NewService(mockRepo, NewCommentPubSub())

	ctx := context.Background()

	tests := []struct {
		name        string
		postID      string
		enabled     bool
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name:    "enable comments",
			postID:  "post1",
			enabled: true,
			mockSetup: func() {
				mockRepo.EXPECT().
					SetCommentsEnabled(gomock.Any(), "post1", true).
					Return(&domain.Post{ID: "post1", CommentsEnabled: true}, nil)
			},
		},
		{
			name:    "disable comments",
			postID:  "post2",
			enabled: false,
			mockSetup: func() {
				mockRepo.EXPECT().
					SetCommentsEnabled(gomock.Any(), "post2", false).
					Return(&domain.Post{ID: "post2", CommentsEnabled: false}, nil)
			},
		},
		{
			name:        "empty post id",
			postID:      "",
			enabled:     true,
			wantErr:     true,
			errContains: "post id must not be empty",
		},
		{
			name:    "repository error",
			postID:  "post3",
			enabled: true,
			mockSetup: func() {
				mockRepo.EXPECT().
					SetCommentsEnabled(gomock.Any(), "post3", true).
					Return(nil, fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to switch comments state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			post, err := s.SetCommentsEnabled(ctx, tt.postID, tt.enabled)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, post)
			assert.Equal(t, tt.postID, post.ID)

			t.Logf("comments state updated for post: %s", post.ID)
		})
	}
}

func TestServiceDeletePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()

	tests := []struct {
		name        string
		id          string
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name: "successful deletion",
			id:   "post1",
			mockSetup: func() {
				mockRepo.EXPECT().
					DeletePost(gomock.Any(), "post1").
					Return(nil)
			},
		},
		{
			name:        "empty post id",
			id:          "",
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "cannot delete post with empty id",
		},
		{
			name: "repo error",
			id:   "post2",
			mockSetup: func() {
				mockRepo.EXPECT().
					DeletePost(gomock.Any(), "post2").
					Return(fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to delete post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := s.DeletePost(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			t.Logf("post deleted successfully: %s", tt.id)
		})
	}
}

func TestServiceCreateComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	s := NewService(mockRepo, NewCommentPubSub())

	ctx := context.Background()

	tests := []struct {
		name        string
		comment     *domain.Comment
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name: "successful creation",
			comment: &domain.Comment{
				AuthorID: "user1",
				PostID:   "post1",
				Content:  "test",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateComment(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, c *domain.Comment) error {
						assert.NotEmpty(t, c.ID)
						assert.False(t, c.CreatedAt.IsZero())
						return nil
					})
			},
		},
		{
			name: "content exceeds max length",
			comment: &domain.Comment{
				AuthorID: "user1",
				PostID:   "post1",
				Content:  strings.Repeat("a", MaxCommentLength+1),
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "comment content exceeds maximum allowed length",
		},
		{
			name: "empty content",
			comment: &domain.Comment{
				AuthorID: "user1",
				PostID:   "post1",
			},
			wantErr:     true,
			errContains: "comment must not be empty",
		},
		{
			name: "empty author id",
			comment: &domain.Comment{
				PostID:  "post1",
				Content: "test",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "author with non-empty id",
		},
		{
			name: "empty author id",
			comment: &domain.Comment{
				PostID:  "post1",
				Content: "test",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "author with non-empty id",
		},
		{
			name: "repository error",
			comment: &domain.Comment{
				AuthorID: "user1",
				PostID:   "post1",
				Content:  "test",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateComment(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to create comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			res, err := s.CreateComment(ctx, tt.comment)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			t.Logf("comment created: %s", res.ID)
		})
	}
}

func TestServiceGetCommentByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()

	tests := []struct {
		name        string
		id          string
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name: "success",
			id:   "c1",
			mockSetup: func() {
				mockRepo.EXPECT().
					GetCommentByID(gomock.Any(), "c1").
					Return(&domain.Comment{ID: "c1"}, nil)
			},
		},
		{
			name:        "empty id",
			id:          "",
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "cannot get comment with empty id",
		},
		{
			name: "repo error",
			id:   "c2",
			mockSetup: func() {
				mockRepo.EXPECT().
					GetCommentByID(gomock.Any(), "c2").
					Return(nil, fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to get comment by id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			comment, err := s.GetCommentByID(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, comment)
			assert.Equal(t, tt.id, comment.ID)

			t.Logf("comment retrieved successfully: %+v", comment)
		})
	}
}

func TestServiceGetComments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()
	parentID := "parent1"

	tests := []struct {
		name        string
		postID      string
		parentID    *string
		limit       int
		offset      int
		order       repository.CommentSortOrder
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name:   "top level comments",
			postID: "post1",
			limit:  10,
			offset: 0,
			order:  repository.CommentSortNewestFirst,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetComments(gomock.Any(), "post1", nil, 10, 0, repository.CommentSortNewestFirst).
					Return([]*domain.Comment{{ID: "c1"}}, nil)
			},
		},
		{
			name:     "child comments force oldest",
			postID:   "post1",
			parentID: &parentID,
			limit:    5,
			offset:   0,
			order:    repository.CommentSortNewestFirst,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetComments(gomock.Any(), "post1", &parentID, 5, 0, repository.CommentSortOldestFirst).
					Return([]*domain.Comment{{ID: "c2"}}, nil)
			},
		},
		{
			name:   "limit exceeds max pagination limit",
			postID: "post1",
			limit:  MaxPaginationLimit + 50,
			offset: 0,
			order:  repository.CommentSortNewestFirst,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetComments(gomock.Any(), "post1", nil, MaxPaginationLimit, 0, repository.CommentSortNewestFirst).
					Return([]*domain.Comment{{ID: "c1"}}, nil)
			},
			wantErr: false,
		},
		{
			name:        "invalid pagination",
			postID:      "post1",
			limit:       0,
			offset:      -1,
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "limit must be positive",
		},
		{
			name:   "repo error",
			postID: "post1",
			limit:  10,
			offset: 0,
			order:  repository.CommentSortNewestFirst,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetComments(gomock.Any(), "post1", nil, 10, 0, repository.CommentSortNewestFirst).
					Return(nil, fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to get comments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			comments, err := s.GetComments(ctx, tt.postID, tt.parentID, tt.limit, tt.offset, tt.order)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, comments)

			t.Logf("comments retrieved successfully: %+v", comments)
		})
	}
}

func TestServiceDeleteComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	s := NewService(mockRepo, NewCommentPubSub())

	ctx := context.Background()

	tests := []struct {
		name        string
		id          string
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name: "success",
			id:   "c1",
			mockSetup: func() {
				mockRepo.EXPECT().
					DeleteComment(gomock.Any(), "c1").
					Return(nil)
			},
		},
		{
			name:        "empty id",
			id:          "",
			wantErr:     true,
			errContains: "cannot delete comment",
		},
		{
			name: "repo error",
			id:   "c2",
			mockSetup: func() {
				mockRepo.EXPECT().
					DeleteComment(gomock.Any(), "c2").
					Return(fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to delete comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			err := s.DeleteComment(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			t.Logf("comment deleted: %s", tt.id)
		})
	}
}
