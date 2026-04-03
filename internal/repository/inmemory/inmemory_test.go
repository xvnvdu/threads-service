package inmemory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xvnvdu/threads-service/internal/domain"
	"github.com/xvnvdu/threads-service/internal/repository"
)

func createTestPost(t *testing.T, r *InMemoryRepository, authorID, title, content string, commentsEnabled bool) *domain.Post {
	post := &domain.Post{
		ID:              uuid.New().String(),
		AuthorID:        authorID,
		Title:           title,
		Content:         content,
		CommentsEnabled: commentsEnabled,
		CreatedAt:       time.Now(),
	}
	t.Logf("attempting to create post with AuthorID: %s, Title: %s", authorID, title)
	err := r.CreatePost(context.Background(), post)
	require.NoError(t, err, "CreatePost in helper should not fail")
	t.Logf("successfully saved post with ID: %s", post.ID)
	return post
}

func createTestComment(t *testing.T, r *InMemoryRepository, postID, authorID, content string, parentID *string) *domain.Comment {
	comment := &domain.Comment{
		ID:        uuid.New().String(),
		PostID:    postID,
		AuthorID:  authorID,
		Content:   content,
		ParentID:  parentID,
		CreatedAt: time.Now(),
	}
	parentIDStr := "nil"
	if parentID != nil {
		parentIDStr = *parentID
	}
	t.Logf("attempting to create comment for PostID: %s, AuthorID: %s, ParentID: %s", postID, authorID, parentIDStr)
	err := r.CreateComment(context.Background(), comment)
	require.NoError(t, err, "CreateComment in helper should not fail")
	t.Logf("successfully saved comment with ID: %s", comment.ID)
	return comment
}

func TestInMemoryRepositoryGetPostByID(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	post1 := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	post2 := createTestPost(t, r, "user2", "Post 2", "Content 2", false)

	tests := []struct {
		name string
		id   string
		want *domain.Post
	}{
		{"Found Post 1", post1.ID, post1},
		{"Found Post 2", post2.ID, post2},
		{"Not Found", "non-existent-id", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("attempting to get post by ID: %s", tt.id)
			got, err := r.GetPostByID(ctx, tt.id)
			require.NoError(t, err, "GetPostByID should not return an error")
			assert.Equal(t, tt.want, got)

			if got != nil {
				t.Logf("successfully retrieved post with ID: %s", got.ID)
			} else {
				t.Log("post not found as expected.")
			}
		})
	}
}

func TestInMemoryRepositoryGetPostsPaginationAndSort(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	t.Log("creating 5 test posts for pagination...")
	// Создаем 5 постов (порядок важен для сортировки)
	time.Sleep(1 * time.Millisecond) // Чтобы CreatedAt были разными
	post5 := createTestPost(t, r, "user", "Post 5", "c", true)
	time.Sleep(1 * time.Millisecond)
	post4 := createTestPost(t, r, "user", "Post 4", "c", true)
	time.Sleep(1 * time.Millisecond)
	post3 := createTestPost(t, r, "user", "Post 3", "c", true)
	time.Sleep(1 * time.Millisecond)
	post2 := createTestPost(t, r, "user", "Post 2", "c", true)
	time.Sleep(1 * time.Millisecond)
	post1 := createTestPost(t, r, "user", "Post 1", "c", true)

	// Ожидаемый порядок (от новых к старым)
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
			got, err := r.GetPosts(ctx, tt.limit, tt.offset)
			require.NoError(t, err, "GetPosts() should not produce an error")
			assert.Equal(t, tt.want, got)
			t.Logf("successfully retrieved %d posts.", len(got))
			for i, p := range got {
				t.Logf("  [%d] Post ID: %s", i, p.ID)
			}
		})
	}
}

func TestInMemoryRepositoryDeletePost(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	post1 := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	t.Logf("created post1 with ID: %s", post1.ID)
	c1 := createTestComment(t, r, post1.ID, "u1", "C1", nil)
	t.Logf("created c1 with ID: %s", c1.ID)
	c1_1 := createTestComment(t, r, post1.ID, "u2", "C1.1", &c1.ID)
	t.Logf("created c1_1 with ID: %s", c1_1.ID)
	c1_1_1 := createTestComment(t, r, post1.ID, "u3", "C1.1.1", &c1_1.ID)
	t.Logf("created c1_1_1 with ID: %s", c1_1_1.ID)
	c1_2 := createTestComment(t, r, post1.ID, "u4", "C1.2", nil)
	t.Logf("created c1_2 with ID: %s", c1_2.ID)

	// Проверяем изначальное состояние
	initialCommentCount := len(r.comments)
	t.Logf("initial comment count: %d", initialCommentCount)
	require.Equal(t, 4, initialCommentCount)

	t.Logf("attempting to delete post with ID: %s", post1.ID)
	err := r.DeletePost(ctx, post1.ID)
	require.NoError(t, err)
	t.Logf("post %s deleted successfully.", post1.ID)

	// Проверяем, что пост удален
	gotPost, err := r.GetPostByID(ctx, post1.ID)
	require.NoError(t, err)
	assert.Nil(t, gotPost, "Post should be deleted")
	t.Log("post successfully verified as deleted.")

	// Проверяем, что комментарии поста удалились вместе с ним
	t.Log("verifying associated comments are deleted...")
	checkCommentDeleted(t, r, c1.ID, "C1")
	checkCommentDeleted(t, r, c1_1.ID, "C1.1")
	checkCommentDeleted(t, r, c1_1_1.ID, "C1.1.1")

	// C1.2 был создан под post1, так что тоже должен быть удален.
	checkCommentDeleted(t, r, c1_2.ID, "C1.2")

	assert.Empty(t, r.comments, "Comments map should be empty after deleting post")
	t.Log("all comments associated with the post verified as deleted.")

	// Удаление несуществующего поста
	nonExistentPostID := "non-existent-id"
	t.Logf("attempting to delete non-existent post ID: %s", nonExistentPostID)
	err = r.DeletePost(ctx, nonExistentPostID)
	require.Error(t, err, "Expected an error for non-existent post")
	t.Logf("correctly received expected error for non-existent post: %v", err)
}

func TestInMemoryRepositoryCreateComment(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()
	post1 := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	post2 := createTestPost(t, r, "user2", "Post 2", "Content 2", false)
	t.Logf("created post1 (ID: %s, CommentsEnabled: %v), post2 (ID: %s, CommentsEnabled: %v)", post1.ID, post1.CommentsEnabled, post2.ID, post2.CommentsEnabled)

	// Успешное создание
	comment1 := &domain.Comment{
		ID:       uuid.New().String(),
		PostID:   post1.ID,
		AuthorID: "user1",
		Content:  "First comment",
	}
	t.Logf("attempting to create comment for post1 (enabled)")
	err := r.CreateComment(ctx, comment1)
	require.NoError(t, err)
	t.Logf("successfully created comment with ID: %s", comment1.ID)

	// Ответ на комментарий
	comment1_1 := &domain.Comment{
		ID:       uuid.New().String(),
		PostID:   post1.ID,
		AuthorID: "user2",
		Content:  "Reply to first comment",
		ParentID: &comment1.ID,
	}
	t.Logf("attempting to create reply for comment %s (PostID: %s)", comment1.ID, post1.ID)
	err = r.CreateComment(ctx, comment1_1)
	require.NoError(t, err)
	t.Logf("successfully created reply with ID: %s", comment1_1.ID)

	// Комментарии отключены
	commentDisabled := &domain.Comment{
		ID:       uuid.New().String(),
		PostID:   post2.ID,
		AuthorID: "user3",
		Content:  "Comment for disabled post",
	}
	t.Logf("attempting to create comment for post2 (disabled)")
	err = r.CreateComment(ctx, commentDisabled)
	require.Error(t, err)
	expectedErr := fmt.Sprintf("post with ID %s does not allow comments", post2.ID)
	assert.EqualError(t, err, expectedErr)
	t.Logf("correctly received expected error for disabled comments: %v", err)

	// Несуществующий пост
	nonExistentPostID := "non-existent"
	commentNonExistentPost := &domain.Comment{
		ID:       uuid.New().String(),
		PostID:   nonExistentPostID,
		AuthorID: "user4",
		Content:  "Comment for non-existent post",
	}
	t.Logf("attempting to create comment for non-existent post ID: %s", nonExistentPostID)
	err = r.CreateComment(ctx, commentNonExistentPost)
	require.Error(t, err)
	expectedErr = fmt.Sprintf("cannot create comment, post with ID %s does not exist", nonExistentPostID)
	assert.EqualError(t, err, expectedErr)
	t.Logf("correctly received expected error for non-existent post: %v", err)

	// Ответ на несуществующий комментарий
	nonExistentParentID := "non-existent-parent"
	commentNonExistentParent := &domain.Comment{
		ID:       uuid.New().String(),
		PostID:   post1.ID,
		AuthorID: "user5",
		Content:  "Reply to non-existent parent",
		ParentID: &nonExistentParentID,
	}
	t.Logf("attempting to create reply to non-existent parent ID: %s", nonExistentParentID)
	err = r.CreateComment(ctx, commentNonExistentParent)
	require.Error(t, err)
	expectedErr = fmt.Sprintf("cannot reply to nonexistent comment with ID %s", nonExistentParentID)
	assert.EqualError(t, err, expectedErr)
	t.Logf("correctly received expected error for non-existent parent: %v", err)
}

func TestInMemoryRepositoryGetCommentByID(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()
	post := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	comment := createTestComment(t, r, post.ID, "user1", "Comment 1", nil)
	t.Logf("created comment with ID: %s", comment.ID)

	tests := []struct {
		name string
		id   string
		want *domain.Comment
	}{
		{"Found Comment", comment.ID, comment},
		{"Not Found", "non-existent-id", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("attempting to get comment by ID: %s", tt.id)
			got, err := r.GetCommentByID(ctx, tt.id)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			if got != nil {
				t.Logf("successfully retrieved comment with ID: %s", got.ID)
			} else {
				t.Log("comment not found as expected.")
			}
		})
	}
}

func checkCommentDeleted(t *testing.T, r *InMemoryRepository, commentID, name string) {
	gotComment, err := r.GetCommentByID(context.Background(), commentID)
	require.NoError(t, err)
	assert.Nil(t, gotComment, fmt.Sprintf("Comment %s (ID: %s) should be deleted", name, commentID))
	t.Logf("comment %s (ID: %s) verified as deleted.", name, commentID)
}

func TestInMemoryRepositoryDeleteComment(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	post1 := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	c1 := createTestComment(t, r, post1.ID, "u1", "C1", nil)
	c1_1 := createTestComment(t, r, post1.ID, "u2", "C1.1", &c1.ID)
	c1_1_1 := createTestComment(t, r, post1.ID, "u3", "C1.1.1", &c1_1.ID)
	c1_2 := createTestComment(t, r, post1.ID, "u4", "C1.2", nil)
	c1_3 := createTestComment(t, r, post1.ID, "u5", "C1.3", nil) // Еще один корневой коммент

	t.Logf("initial comments: C1(%s), C1.1(%s), C1.1.1(%s), C1.2(%s), C1.3(%s)", c1.ID, c1_1.ID, c1_1_1.ID, c1_2.ID, c1_3.ID)

	// Проверяем изначальное состояние
	initialCommentCount := len(r.comments)
	t.Logf("initial total comment count in map: %d", initialCommentCount)
	require.Equal(t, 5, initialCommentCount)

	// Удаляем корневой комментарий C1 (должен удалить C1, C1.1, C1.1.1)
	t.Logf("attempting to delete comment ID: %s (C1) and its branch...", c1.ID)
	err := r.DeleteComment(ctx, c1.ID)
	require.NoError(t, err)
	t.Logf("deletion of C1 branch (ID: %s) completed.", c1.ID)

	// Проверяем, что удалены 3 комментария
	assert.Len(t, r.comments, 2, "Expected 2 comments after deleting C1 branch")
	t.Logf("remaining comment count verified: %d", len(r.comments))

	checkCommentDeleted(t, r, c1.ID, "C1")
	checkCommentDeleted(t, r, c1_1.ID, "C1.1")
	checkCommentDeleted(t, r, c1_1_1.ID, "C1.1.1")

	t.Log("verifying C1.2 and C1.3 still exist...")
	assert.NotNil(t, r.comments[c1_2.ID], "C1.2 should NOT be deleted")
	t.Logf("C1.2 (ID: %s) still exists.", c1_2.ID)
	assert.NotNil(t, r.comments[c1_3.ID], "C1.3 should NOT be deleted")
	t.Logf("C1.3 (ID: %s) still exists.", c1_3.ID)

	// Проверяем, что индекс тоже корректен
	t.Log("verifying commentsIndex state after deletion...")
	_, ok := r.commentsIndex[post1.ID][c1.ID] // c1 был родителем
	assert.False(t, ok, "C1 should be removed as a parent from index")
	t.Logf("C1 (ID: %s) successfully removed as a parent from index.", c1.ID)

	// C1.2 и C1.3 все еще корневые
	rootComments := r.commentsIndex[post1.ID][""]
	assert.Len(t, rootComments, 2, "Should be 2 root comments left")

	// Удаление несуществующего комментария
	nonExistentCommentID := "non-existent-id"
	t.Logf("attempting to delete non-existent comment ID: %s", nonExistentCommentID)
	err = r.DeleteComment(ctx, nonExistentCommentID)
	require.Error(t, err)
	t.Logf("correctly received expected error for non-existent comment: %v", err)
}

func TestInMemoryRepositorySetCommentsEnabled(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()
	post := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	t.Logf("initial post ID: %s, CommentsEnabled: %v", post.ID, post.CommentsEnabled)

	// Выключаем комменты
	t.Logf("attempting to disable comments for post ID: %s", post.ID)
	updatedPost, err := r.SetCommentsEnabled(ctx, post.ID, false)
	require.NoError(t, err)
	require.NotNil(t, updatedPost)
	assert.False(t, updatedPost.CommentsEnabled, "Expected comments to be disabled")
	t.Logf("comments successfully disabled. Updated Post CommentsEnabled: %v", updatedPost.CommentsEnabled)

	// Включаем комменты
	t.Logf("attempting to enable comments for post ID: %s", post.ID)
	updatedPost, err = r.SetCommentsEnabled(ctx, post.ID, true)
	require.NoError(t, err)
	require.NotNil(t, updatedPost)
	assert.True(t, updatedPost.CommentsEnabled, "Expected comments to be enabled")
	t.Logf("comments successfully enabled. Updated Post CommentsEnabled: %v", updatedPost.CommentsEnabled)

	// Пытаемся переключить комменты на несуществующем посте
	nonExistentID := "non-existent-id"
	t.Logf("attempting to set comments enabled for non-existent post ID: %s", nonExistentID)
	_, err = r.SetCommentsEnabled(ctx, nonExistentID, true)
	require.Error(t, err, "Expected error for non-existent post")
	t.Logf("correctly received expected error for non-existent post: %v", err)
}

func TestInMemoryRepositoryGetComments(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	t.Log("creating test post...")
	post := createTestPost(t, r, "user1", "Post", "Content", true)

	t.Log("creating comments...")
	time.Sleep(1 * time.Millisecond)
	c1 := createTestComment(t, r, post.ID, "u1", "C1", nil)

	time.Sleep(1 * time.Millisecond)
	c2 := createTestComment(t, r, post.ID, "u2", "C2", nil)

	time.Sleep(1 * time.Millisecond)
	c3 := createTestComment(t, r, post.ID, "u3", "C3", nil)

	t.Log("creating nested comments...")
	time.Sleep(1 * time.Millisecond)
	c1_1 := createTestComment(t, r, post.ID, "u4", "C1.1", &c1.ID)

	time.Sleep(1 * time.Millisecond)
	c1_2 := createTestComment(t, r, post.ID, "u5", "C1.2", &c1.ID)

	tests := []struct {
		name      string
		postID    string
		parentID  *string
		limit     int
		offset    int
		order     repository.CommentSortOrder
		wantIDs   []string
		wantError bool
	}{
		{
			name:      "Post does not exist",
			postID:    "non-existent",
			parentID:  nil,
			limit:     10,
			offset:    0,
			order:     repository.CommentSortNewestFirst,
			wantError: true,
		},
		{
			name:     "Get root comments newest first",
			postID:   post.ID,
			parentID: nil,
			limit:    10,
			offset:   0,
			order:    repository.CommentSortNewestFirst,
			wantIDs:  []string{c3.ID, c2.ID, c1.ID},
		},
		{
			name:     "Get root comments oldest first",
			postID:   post.ID,
			parentID: nil,
			limit:    10,
			offset:   0,
			order:    repository.CommentSortOldestFirst,
			wantIDs:  []string{c1.ID, c2.ID, c3.ID},
		},
		{
			name:     "Get child comments (forced oldest)",
			postID:   post.ID,
			parentID: &c1.ID,
			limit:    10,
			offset:   0,
			order:    repository.CommentSortOldestFirst,
			wantIDs:  []string{c1_1.ID, c1_2.ID},
		},
		{
			name:     "Pagination first item",
			postID:   post.ID,
			parentID: nil,
			limit:    1,
			offset:   0,
			order:    repository.CommentSortNewestFirst,
			wantIDs:  []string{c3.ID},
		},
		{
			name:     "Pagination second item",
			postID:   post.ID,
			parentID: nil,
			limit:    1,
			offset:   1,
			order:    repository.CommentSortNewestFirst,
			wantIDs:  []string{c2.ID},
		},
		{
			name:     "Offset out of range",
			postID:   post.ID,
			parentID: nil,
			limit:    10,
			offset:   100,
			order:    repository.CommentSortNewestFirst,
			wantIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("getting comments: postID=%s, parentID=%v, limit=%d, offset=%d",
				tt.postID, tt.parentID, tt.limit, tt.offset)

			got, err := r.GetComments(ctx, tt.postID, tt.parentID, tt.limit, tt.offset, tt.order)

			if tt.wantError {
				require.Error(t, err)
				t.Logf("received expected error: %v", err)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, len(tt.wantIDs))

			gotIDs := make([]string, len(got))
			for i, c := range got {
				gotIDs[i] = c.ID
			}

			assert.Equal(t, tt.wantIDs, gotIDs, "Retrieved comment IDs should match expected IDs in order")

			t.Logf("successfully retrieved %d comments", len(got))
			for i, c := range got {
				t.Logf("  [%d] Comment ID: %s", i, c.ID)
			}
		})
	}
}
