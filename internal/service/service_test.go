package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xvnvdu/threads-service/internal/domain"
	"github.com/xvnvdu/threads-service/internal/repository"
	"github.com/xvnvdu/threads-service/internal/repository/mocks"
	"go.uber.org/mock/gomock"
)

func TestServiceCreatePost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()

	tests := []struct {
		name        string
		post        *domain.Post
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name: "Successful creation with empty ID and CreatedAt",
			post: &domain.Post{
				AuthorID: "user1",
				Title:    "Test Title",
				Content:  "Test Content",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreatePost(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, post *domain.Post) error {
						t.Logf("Mock CreatePost called with ID: %s", post.ID)
						if post.ID == "" {
							t.Error("Expected ID to be set")
						}
						if post.CreatedAt.IsZero() {
							t.Error("Expected CreatedAt to be set")
						}
						return nil
					})
			},
			wantErr: false,
		},
		{
			name: "Empty AuthorID",
			post: &domain.Post{
				Title:   "Test Title",
				Content: "Test Content",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "author with empty id",
		},
		{
			name: "Empty Title",
			post: &domain.Post{
				AuthorID: "user1",
				Content:  "Test Content",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "without a title",
		},
		{
			name: "Empty Content",
			post: &domain.Post{
				AuthorID: "user1",
				Title:    "Test Title",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "empty post",
		},
		{
			name: "Repository error",
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
			name: "ID and CreatedAt already set",
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
					DoAndReturn(func(ctx context.Context, post *domain.Post) error {
						t.Logf("Mock CreatePost called with existing ID: %s", post.ID)
						return nil
					})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			result, err := s.CreatePost(ctx, tt.post)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				t.Logf("Received expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}
			t.Logf("Post successfully created with ID: %s", result.ID)
		})
	}
}

func TestServiceGetPostsAndGetPostByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()

	testsGetPostByID := []struct {
		name        string
		id          string
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name: "Successful retrieval",
			id:   "post1",
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPostByID(gomock.Any(), "post1").
					Return(&domain.Post{ID: "post1", Title: "Title"}, nil)
			},
			wantErr: false,
		},
		{
			name: "Repository returns error",
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

	for _, tt := range testsGetPostByID {
		t.Run("GetPostByID: "+tt.name, func(t *testing.T) {
			tt.mockSetup()

			post, err := s.GetPostByID(ctx, tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				t.Logf("Received expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if post == nil || post.ID != tt.id {
				t.Fatalf("Expected post ID '%s', got %+v", tt.id, post)
			}
			t.Logf("Post retrieved successfully: %+v", post)
		})
	}

	testsGetPosts := []struct {
		name        string
		limit       int
		offset      int
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name:   "Successful retrieval",
			limit:  10,
			offset: 0,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPosts(gomock.Any(), 10, 0).
					Return([]*domain.Post{
						{ID: "p1"}, {ID: "p2"},
					}, nil)
			},
			wantErr: false,
		},
		{
			name:        "Invalid limit",
			limit:       0,
			offset:      0,
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "limit must be positive",
		},
		{
			name:        "Invalid offset",
			limit:       5,
			offset:      -1,
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "offset must be non-negative",
		},
		{
			name:   "Repository returns error",
			limit:  5,
			offset: 0,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPosts(gomock.Any(), 5, 0).
					Return(nil, fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to get posts",
		},
		{
			name:   "Limit exceeds MaxPaginationLimit",
			limit:  MaxPaginationLimit + 100,
			offset: 0,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetPosts(gomock.Any(), MaxPaginationLimit, 0).
					Return([]*domain.Post{{ID: "p1"}}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range testsGetPosts {
		t.Run("GetPosts: "+tt.name, func(t *testing.T) {
			tt.mockSetup()

			posts, err := s.GetPosts(ctx, tt.limit, tt.offset)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				t.Logf("Received expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(posts) == 0 {
				t.Fatal("Expected at least one post, got empty slice")
			}
			t.Logf("Posts retrieved successfully: %+v", posts)
		})
	}
}

func TestServiceSetCommentsEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

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
			name:    "Successful enable comments",
			postID:  "post1",
			enabled: true,
			mockSetup: func() {
				mockRepo.EXPECT().
					SetCommentsEnabled(gomock.Any(), "post1", true).
					Return(&domain.Post{ID: "post1", Title: "Title"}, nil)
			},
			wantErr: false,
		},
		{
			name:    "Successful disable comments",
			postID:  "post2",
			enabled: false,
			mockSetup: func() {
				mockRepo.EXPECT().
					SetCommentsEnabled(gomock.Any(), "post2", false).
					Return(&domain.Post{ID: "post2", Title: "Another"}, nil)
			},
			wantErr: false,
		},
		{
			name:        "Empty post ID",
			postID:      "",
			enabled:     true,
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "post id must not be empty",
		},
		{
			name:    "Repository returns error",
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
			tt.mockSetup()

			post, err := s.SetCommentsEnabled(ctx, tt.postID, tt.enabled)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				t.Logf("Received expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if post == nil || post.ID != tt.postID {
				t.Fatalf("Expected post ID '%s', got %+v", tt.postID, post)
			}
			t.Logf("Comments state successfully switched for post: %+v", post)
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
			name: "Successful deletion",
			id:   "post1",
			mockSetup: func() {
				mockRepo.EXPECT().
					DeletePost(gomock.Any(), "post1").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "Empty post ID",
			id:          "",
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "cannot delete post with empty id",
		},
		{
			name: "Repository returns error",
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
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				t.Logf("Received expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			t.Logf("Post with ID '%s' successfully deleted", tt.id)
		})
	}
}

func TestServiceCreateComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()

	tests := []struct {
		name        string
		comment     *domain.Comment
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name: "Successful creation with empty ID and CreatedAt",
			comment: &domain.Comment{
				AuthorID: "user1",
				PostID:   "post1",
				Content:  "Test Comment",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateComment(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, comment *domain.Comment) error {
						t.Logf("Mock CreateComment called with ID: %s", comment.ID)
						if comment.ID == "" {
							t.Error("Expected ID to be set")
						}
						if comment.CreatedAt.IsZero() {
							t.Error("Expected CreatedAt to be set")
						}
						return nil
					})
			},
			wantErr: false,
		},
		{
			name: "Empty content",
			comment: &domain.Comment{
				AuthorID: "user1",
				PostID:   "post1",
				Content:  "",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "comment must not be empty",
		},
		{
			name: "Content exceeds MaxCommentLength",
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
			name: "Empty AuthorID",
			comment: &domain.Comment{
				PostID:  "post1",
				Content: "Test",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "comment must have its author with non-empty id",
		},
		{
			name: "Empty PostID",
			comment: &domain.Comment{
				AuthorID: "user1",
				Content:  "Test",
			},
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "comment must be created under a post with non-empty id",
		},
		{
			name: "Repository returns error",
			comment: &domain.Comment{
				AuthorID: "user1",
				PostID:   "post1",
				Content:  "Test Comment",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateComment(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to create comment",
		},
		{
			name: "ID and CreatedAt already set",
			comment: &domain.Comment{
				ID:        uuid.New().String(),
				CreatedAt: time.Now(),
				AuthorID:  "user1",
				PostID:    "post1",
				Content:   "Test Comment",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateComment(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, comment *domain.Comment) error {
						t.Logf("Mock CreateComment called with existing ID: %s", comment.ID)
						return nil
					})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			result, err := s.CreateComment(ctx, tt.comment)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				t.Logf("Received expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if result.ID == "" || result.CreatedAt.IsZero() {
				t.Fatalf("Expected ID and CreatedAt to be set, got ID='%s', CreatedAt='%v'", result.ID, result.CreatedAt)
			}

			t.Logf("Comment successfully created with ID: %s", result.ID)
		})
	}
}

func TestServiceGetCommentsAndGetCommentsByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()

	parentID := "parent1"

	tests := []struct {
		name        string
		getComment  bool // true: GetCommentByID, false: GetComments
		commentID   string
		postID      string
		parentID    *string
		limit       int
		offset      int
		order       repository.CommentSortOrder
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		// GetCommentByID
		{
			name:       "GetCommentByID successful",
			getComment: true,
			commentID:  "c1",
			mockSetup: func() {
				mockRepo.EXPECT().
					GetCommentByID(gomock.Any(), "c1").
					Return(&domain.Comment{ID: "c1", Content: "Test"}, nil)
			},
			wantErr: false,
		},
		{
			name:        "GetCommentByID empty ID",
			getComment:  true,
			commentID:   "",
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "cannot get comment with empty id",
		},
		{
			name:       "GetCommentByID repo error",
			getComment: true,
			commentID:  "c2",
			mockSetup: func() {
				mockRepo.EXPECT().
					GetCommentByID(gomock.Any(), "c2").
					Return(nil, fmt.Errorf("db error"))
			},
			wantErr:     true,
			errContains: "failed to get comment by id",
		},

		// GetComments
		{
			name:       "GetComments successful, top-level comments",
			getComment: false,
			postID:     "post1",
			parentID:   nil,
			limit:      10,
			offset:     0,
			order:      repository.CommentSortNewestFirst,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetComments(gomock.Any(), "post1", nil, 10, 0, repository.CommentSortNewestFirst).
					Return([]*domain.Comment{{ID: "c1"}}, nil)
			},
			wantErr: false,
		},
		{
			name:       "GetComments parentID present, order overridden to OldestFirst",
			getComment: false,
			postID:     "post1",
			parentID:   &parentID,
			limit:      5,
			offset:     0,
			order:      repository.CommentSortNewestFirst,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetComments(gomock.Any(), "post1", &parentID, 5, 0, repository.CommentSortOldestFirst).
					Return([]*domain.Comment{{ID: "c2"}}, nil)
			},
			wantErr: false,
		},
		{
			name:        "GetComments negative offset",
			getComment:  false,
			postID:      "post1",
			parentID:    nil,
			limit:       10,
			offset:      -1,
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "limit must be positive and offset must be non-negative",
		},
		{
			name:        "GetComments zero limit",
			getComment:  false,
			postID:      "post1",
			parentID:    nil,
			limit:       0,
			offset:      0,
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "limit must be positive and offset must be non-negative",
		},
		{
			name:       "GetComments limit exceeds MaxPaginationLimit",
			getComment: false,
			postID:     "post1",
			parentID:   nil,
			limit:      MaxPaginationLimit + 50,
			offset:     0,
			order:      repository.CommentSortNewestFirst,
			mockSetup: func() {
				mockRepo.EXPECT().
					GetComments(gomock.Any(), "post1", nil, MaxPaginationLimit, 0, repository.CommentSortNewestFirst).
					Return([]*domain.Comment{{ID: "c3"}}, nil)
			},
			wantErr: false,
		},
		{
			name:       "GetComments repo error",
			getComment: false,
			postID:     "post1",
			parentID:   nil,
			limit:      10,
			offset:     0,
			order:      repository.CommentSortNewestFirst,
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

			if tt.getComment {
				comment, err := s.GetCommentByID(ctx, tt.commentID)
				if tt.wantErr {
					if err == nil {
						t.Fatalf("Expected error, got nil")
					}
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
					}
					t.Logf("Received expected error: %v", err)
					return
				}

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if comment == nil || comment.ID != tt.commentID {
					t.Fatalf("Expected comment ID '%s', got %+v", tt.commentID, comment)
				}
				t.Logf("Comment successfully retrieved: %+v", comment)

			} else {
				comments, err := s.GetComments(ctx, tt.postID, tt.parentID, tt.limit, tt.offset, tt.order)
				if tt.wantErr {
					if err == nil {
						t.Fatalf("Expected error, got nil")
					}
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
					}
					t.Logf("Received expected error: %v", err)
					return
				}

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if comments == nil {
					t.Fatal("Expected comments slice, got nil")
				}
				t.Logf("Comments successfully retrieved: %+v", comments)
			}
		})
	}
}

func TestServiceDeleteComment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()

	tests := []struct {
		name        string
		commentID   string
		mockSetup   func()
		wantErr     bool
		errContains string
	}{
		{
			name:      "Successful deletion",
			commentID: "c1",
			mockSetup: func() {
				mockRepo.EXPECT().
					DeleteComment(gomock.Any(), "c1").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "Empty comment ID",
			commentID:   "",
			mockSetup:   func() {},
			wantErr:     true,
			errContains: "cannot delete comment with empty id",
		},
		{
			name:      "Repository returns error",
			commentID: "c2",
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
			tt.mockSetup()

			err := s.DeleteComment(ctx, tt.commentID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				t.Logf("Received expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			t.Logf("Comment successfully deleted with ID: %s", tt.commentID)
		})
	}
}
