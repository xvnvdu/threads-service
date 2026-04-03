package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xvnvdu/threads-service/internal/domain"
	"github.com/xvnvdu/threads-service/internal/repository"

	_ "github.com/lib/pq"
)

const testConnString = "postgres://threads_user:12345@localhost:5432/threads_test_db?sslmode=disable"

func newTestDB(t *testing.T) (*PostgresRepository, func()) {
	t.Helper()

	repo, err := NewPostgresRepository(testConnString)
	require.NoError(t, err)

	cleanup := func() {
		_, err := repo.db.Exec("TRUNCATE TABLE posts, comments RESTART IDENTITY CASCADE")
		require.NoError(t, err)
	}

	cleanup()

	return repo, func() {
		repo.Close()
	}
}

func createTestPost(t *testing.T, r repository.Repository, authorID, title, content string, commentsEnabled bool) *domain.Post {
	t.Helper()

	post := &domain.Post{
		ID:              uuid.New().String(),
		AuthorID:        authorID,
		Title:           title,
		Content:         content,
		CommentsEnabled: commentsEnabled,
		CreatedAt:       time.Now().UTC().Truncate(time.Millisecond),
	}
	t.Logf("creating post with ID: %s", post.ID)

	err := r.CreatePost(context.Background(), post)
	require.NoError(t, err)

	return post
}

func createTestComment(t *testing.T, r repository.Repository, postID, authorID, content string, parentID *string) *domain.Comment {
	t.Helper()

	comment := &domain.Comment{
		ID:        uuid.New().String(),
		PostID:    postID,
		AuthorID:  authorID,
		Content:   content,
		ParentID:  parentID,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	t.Logf("creating comment with ID: %s, postID: %s, parentID: %v", comment.ID, postID, parentID)

	err := r.CreateComment(context.Background(), comment)
	require.NoError(t, err)

	return comment
}

func TestPostgresRepositoryGetPostByID(t *testing.T) {
	repo, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	existingPost := createTestPost(
		t,
		repo,
		"author-1",
		"title-1",
		"content-1",
		true,
	)

	tests := []struct {
		name        string
		postID      string
		expectFound bool
	}{
		{
			name:        "post exists",
			postID:      existingPost.ID,
			expectFound: true,
		},
		{
			name:        "post does not exist",
			postID:      uuid.New().String(),
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("getting post ID: %s", tt.postID)

			post, err := repo.GetPostByID(ctx, tt.postID)

			require.NoError(t, err)

			if tt.expectFound {
				require.NotNil(t, post)

				assert.Equal(t, existingPost.ID, post.ID)
				assert.Equal(t, existingPost.Title, post.Title)
				assert.Equal(t, existingPost.Content, post.Content)
				assert.Equal(t, existingPost.AuthorID, post.AuthorID)
				assert.Equal(t, existingPost.CommentsEnabled, post.CommentsEnabled)

				assert.WithinDuration(t, existingPost.CreatedAt, post.CreatedAt, time.Second)
				return
			}

			assert.Nil(t, post)
		})
	}
}

func TestPostgresRepositoryGetPosts(t *testing.T) {
	repo, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Log("creating 5 test posts with distinct CreatedAt...")
	post5 := createTestPost(t, repo, "user", "Post 5", "c", true)
	time.Sleep(1 * time.Millisecond)
	post4 := createTestPost(t, repo, "user", "Post 4", "c", true)
	time.Sleep(1 * time.Millisecond)
	post3 := createTestPost(t, repo, "user", "Post 3", "c", true)
	time.Sleep(1 * time.Millisecond)
	post2 := createTestPost(t, repo, "user", "Post 2", "c", true)
	time.Sleep(1 * time.Millisecond)
	post1 := createTestPost(t, repo, "user", "Post 1", "c", true)

	expectedPosts := []*domain.Post{post1, post2, post3, post4, post5}
	t.Logf("posts created with IDs (newest to oldest): %s, %s, %s, %s, %s",
		post1.ID, post2.ID, post3.ID, post4.ID, post5.ID)

	tests := []struct {
		name   string
		limit  int
		offset int
		want   []*domain.Post
	}{
		{"First 2 posts", 2, 0, expectedPosts[0:2]},
		{"Next 2 posts", 2, 2, expectedPosts[2:4]},
		{"Last post", 2, 4, expectedPosts[4:5]},
		{"All posts", 10, 0, expectedPosts},
		{"Large limit", 100, 0, expectedPosts},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("getting posts with Limit: %d, Offset: %d", tt.limit, tt.offset)

			got, err := repo.GetPosts(ctx, tt.limit, tt.offset)
			require.NoError(t, err, "GetPosts() should not produce an error")

			assert.Len(t, got, len(tt.want))

			for i := range got {
				assert.Equal(t, tt.want[i].ID, got[i].ID)
				assert.Equal(t, tt.want[i].Title, got[i].Title)
				assert.Equal(t, tt.want[i].Content, got[i].Content)
				assert.Equal(t, tt.want[i].AuthorID, got[i].AuthorID)
				assert.Equal(t, tt.want[i].CommentsEnabled, got[i].CommentsEnabled)
				assert.WithinDuration(t, tt.want[i].CreatedAt, got[i].CreatedAt, time.Millisecond)
			}

			t.Logf("successfully retrieved %d posts.", len(got))
		})
	}
}

func TestPostgresRepositorySetCommentsEnabled(t *testing.T) {
	repo, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Log("creating test post...")
	post := createTestPost(t, repo, "user", "Test Post", "content", true)

	tests := []struct {
		name        string
		postID      string
		enabled     bool
		expectError bool
	}{
		{"Disable comments", post.ID, false, false},
		{"Enable comments again", post.ID, true, false},
		{"Non-existent post", "non-existent-id", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("setting CommentsEnabled: %v for postID: %s", tt.enabled, tt.postID)
			got, err := repo.SetCommentsEnabled(ctx, tt.postID, tt.enabled)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, post.ID, got.ID)
			assert.Equal(t, post.Title, got.Title)
			assert.Equal(t, post.Content, got.Content)
			assert.Equal(t, post.AuthorID, got.AuthorID)
			assert.Equal(t, tt.enabled, got.CommentsEnabled)
			assert.WithinDuration(t, post.CreatedAt, got.CreatedAt, time.Millisecond)

			dbPost, err := repo.GetPostByID(ctx, post.ID)
			require.NoError(t, err)
			require.NotNil(t, dbPost)
			assert.Equal(t, tt.enabled, dbPost.CommentsEnabled)
		})
	}
}

func TestPostgresRepositoryDeletePost(t *testing.T) {
	repo, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Log("creating test post...")
	post := createTestPost(t, repo, "user", "Test Post", "content", true)

	tests := []struct {
		name        string
		postID      string
		expectError bool
	}{
		{"Delete existing post", post.ID, false},
		{"Delete non-existent post", "non-existent-id", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("deleting post with ID: %s", tt.postID)
			err := repo.DeletePost(ctx, tt.postID)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			got, err := repo.GetPostByID(ctx, tt.postID)
			require.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}

func TestPostgresRepositoryGetCommentByID(t *testing.T) {
	repo, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Log("creating test post and comment...")
	post := createTestPost(t, repo, "user", "Post for comment", "content", true)
	comment := createTestComment(t, repo, post.ID, "commenter", "Hello, world!", nil)

	tests := []struct {
		name        string
		commentID   string
		expectFound bool
	}{
		{"Comment exists", comment.ID, true},
		{"Comment does not exist", uuid.New().String(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("getting comment with ID: %s", tt.commentID)
			got, err := repo.GetCommentByID(ctx, tt.commentID)
			require.NoError(t, err)

			if !tt.expectFound {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, comment.ID, got.ID)
			assert.Equal(t, comment.PostID, got.PostID)
			if comment.ParentID != nil {
				assert.NotNil(t, got.ParentID)
				assert.Equal(t, *comment.ParentID, *got.ParentID)
			} else {
				assert.Nil(t, got.ParentID)
			}
			assert.Equal(t, comment.Content, got.Content)
			assert.Equal(t, comment.AuthorID, got.AuthorID)
			assert.WithinDuration(t, comment.CreatedAt, got.CreatedAt, time.Millisecond)
		})
	}
}

func TestPostgresRepositoryGetComments(t *testing.T) {
	repo, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Log("creating test post and comments...")
	post := createTestPost(t, repo, "author1", "Post for comments", "content", true)

	// root комментарии
	time.Sleep(1 * time.Millisecond)
	root1 := createTestComment(t, repo, post.ID, "user1", "Root comment 1", nil)
	time.Sleep(1 * time.Millisecond)
	root2 := createTestComment(t, repo, post.ID, "user2", "Root comment 2", nil)

	// Ответы на root1
	time.Sleep(1 * time.Millisecond)
	reply1 := createTestComment(t, repo, post.ID, "user3", "Reply to root1_1", &root1.ID)
	time.Sleep(1 * time.Millisecond)
	reply2 := createTestComment(t, repo, post.ID, "user4", "Reply to root1_2", &root1.ID)

	tests := []struct {
		name     string
		parentID *string
		order    repository.CommentSortOrder
		limit    int
		offset   int
		expected []*domain.Comment
	}{
		{
			"Get root comments newest first",
			nil,
			repository.CommentSortNewestFirst,
			10,
			0,
			[]*domain.Comment{root2, root1},
		},
		{
			"Get root comments oldest first",
			nil,
			repository.CommentSortOldestFirst,
			10,
			0,
			[]*domain.Comment{root1, root2},
		},
		{
			"Get replies to root1 newest first",
			&root1.ID,
			repository.CommentSortNewestFirst,
			10,
			0,
			[]*domain.Comment{reply2, reply1},
		},
		{
			"Get replies to root1 oldest first",
			&root1.ID,
			repository.CommentSortOldestFirst,
			10,
			0,
			[]*domain.Comment{reply1, reply2},
		},
		{
			"Pagination test limit 1",
			nil,
			repository.CommentSortOldestFirst,
			1,
			0,
			[]*domain.Comment{root1},
		},
		{
			"Pagination test limit 1 offset 1",
			nil,
			repository.CommentSortOldestFirst,
			1,
			1,
			[]*domain.Comment{root2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetComments(ctx, post.ID, tt.parentID, tt.limit, tt.offset, tt.order)
			require.NoError(t, err)
			require.Len(t, got, len(tt.expected))

			for i, c := range got {
				assert.Equal(t, tt.expected[i].ID, c.ID)
				assert.Equal(t, tt.expected[i].PostID, c.PostID)
				assert.Equal(t, tt.expected[i].Content, c.Content)
				assert.Equal(t, tt.expected[i].AuthorID, c.AuthorID)
				if tt.expected[i].ParentID != nil {
					assert.NotNil(t, c.ParentID)
					assert.Equal(t, *tt.expected[i].ParentID, *c.ParentID)
				} else {
					assert.Nil(t, c.ParentID)
				}
				assert.WithinDuration(t, tt.expected[i].CreatedAt, c.CreatedAt, time.Millisecond)
			}
		})
	}
}

func TestPostgresRepositoryDeleteComment(t *testing.T) {
	repo, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Log("creating test post and comment...")
	post := createTestPost(t, repo, "user", "Post for comment", "content", true)
	comment := createTestComment(t, repo, post.ID, "commenter", "Hello there", nil)

	tests := []struct {
		name        string
		commentID   string
		expectError bool
	}{
		{"Delete existing comment", comment.ID, false},
		{"Delete non-existent comment", comment.ID, true}, // Возьмем прям тот же удаленный
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("deleting comment with ID: %s", tt.commentID)
			err := repo.DeleteComment(ctx, tt.commentID)

			if !tt.expectError {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)

			got, err := repo.GetCommentByID(ctx, tt.commentID)
			require.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}

// Отдельно проверяем что комменты удаляются автоматом при удалении родителя/поста
func TestPostgresRepositoryOnDeleteCascade(t *testing.T) {
	repo, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Log("creating test post...")
	post := createTestPost(t, repo, "user1", "Post for deleting comments", "Post content", true)

	t.Log("creating test root comments...")
	root1 := createTestComment(t, repo, post.ID, "user2", "Root comment 1", nil)
	root2 := createTestComment(t, repo, post.ID, "user3", "Root comment 2", nil)

	t.Log("creating test replies...")
	reply1 := createTestComment(t, repo, post.ID, "user4", "Reply to root1", &root1.ID)
	reply2 := createTestComment(t, repo, post.ID, "user5", "Reply to root1", &root1.ID)

	tests := []struct {
		name        string
		deleteObj   string
		expectError bool
	}{
		{"Delete root comment with replies", "comment", false},
		{"Delete post with comments", "post", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.deleteObj {
			case "comment":
				t.Logf("deleting root comment with ID: %s", root1.ID)
				err := repo.DeleteComment(ctx, root1.ID)
				require.NoError(t, err)

				t.Log("trying to get deleted replies by ID...")
				deletedReply1, err := repo.GetCommentByID(ctx, reply1.ID)
				require.NoError(t, err)
				assert.Nil(t, deletedReply1)

				deletedReply2, err := repo.GetCommentByID(ctx, reply2.ID)
				require.NoError(t, err)
				assert.Nil(t, deletedReply2)

			case "post":
				t.Logf("deleting post with ID: %s", post.ID)
				err := repo.DeletePost(ctx, post.ID)
				require.NoError(t, err)

				t.Log("trying to get comment on deleted post...")
				deletedRoot2, err := repo.GetCommentByID(ctx, root2.ID)
				require.NoError(t, err)
				assert.Nil(t, deletedRoot2)
			}
		})
	}
}
