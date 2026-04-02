package inmemory

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
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
	t.Logf("Attempting to create post with AuthorID: %s, Title: %s", authorID, title)
	err := r.CreatePost(context.Background(), post)
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	t.Logf("Successfully saved post with ID: %s", post.ID)
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
	t.Logf("Attempting to create comment for PostID: %s, AuthorID: %s, ParentID: %s", postID, authorID, parentIDStr)
	err := r.CreateComment(context.Background(), comment)
	if err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}
	t.Logf("Successfully saved comment with ID: %s", comment.ID)
	return comment
}

func TestInMemoryRepositoryGetPostByID(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	post1 := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	post2 := createTestPost(t, r, "user2", "Post 2", "Content 2", false)

	tests := []struct {
		name    string
		id      string
		want    *domain.Post
		wantErr bool
	}{
		{"Found Post 1", post1.ID, post1, false},
		{"Found Post 2", post2.ID, post2, false},
		{"Not Found", "non-existent-id", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Attempting to get post by ID: %s", tt.id)
			got, err := r.GetPostByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPostByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPostByID() got = %v, want %v", got, tt.want)
			} else {
				if got != nil {
					t.Logf("Successfully retrieved post with ID: %s", got.ID)
				} else {
					t.Log("Post not found as expected.")
				}
			}
		})
	}
}

func TestInMemoryRepositoryGetPostsPaginationAndSort(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	t.Log("Creating 5 test posts for pagination...")
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
	t.Logf("Posts created with IDs (newest to oldest): %s, %s, %s, %s, %s",
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
			t.Logf("Getting posts with Limit: %d, Offset: %d", tt.limit, tt.offset)
			got, err := r.GetPosts(ctx, tt.limit, tt.offset)
			if err != nil {
				t.Errorf("GetPosts() unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				// Используем вспомогательную функцию для сравнения слайсов постов,
				// так как DeepEqual может не сработать из-за полей CreatedAt.
				// Проверяем ID и порядок.
				if len(got) != len(tt.want) {
					t.Errorf("GetPosts() got len %d, want len %d", len(got), len(tt.want))
				}
				for i := range got {
					if i >= len(tt.want) { // Избегаем паники при несовпадении длин
						break
					}
					if got[i].ID != tt.want[i].ID {
						t.Errorf("GetPosts() at index %d got ID %s, want ID %s", i, got[i].ID, tt.want[i].ID)
					}
				}
			} else {
				t.Logf("Successfully retrieved %d posts.", len(got))
				for i, p := range got {
					t.Logf("  [%d] Post ID: %s", i, p.ID)
				}
			}
		})
	}
}

func TestInMemoryRepositoryDeletePost(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	post1 := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	t.Logf("Created post1 with ID: %s", post1.ID)
	c1 := createTestComment(t, r, post1.ID, "u1", "C1", nil)
	t.Logf("Created c1 with ID: %s", c1.ID)
	c1_1 := createTestComment(t, r, post1.ID, "u2", "C1.1", &c1.ID)
	t.Logf("Created c1_1 with ID: %s", c1_1.ID)
	c1_1_1 := createTestComment(t, r, post1.ID, "u3", "C1.1.1", &c1_1.ID)
	t.Logf("Created c1_1_1 with ID: %s", c1_1_1.ID)
	c1_2 := createTestComment(t, r, post1.ID, "u4", "C1.2", nil)
	t.Logf("Created c1_2 with ID: %s", c1_2.ID)

	// Проверяем изначальное состояние
	initialCommentCount := len(r.comments)
	t.Logf("Initial comment count: %d", initialCommentCount)
	if initialCommentCount != 4 {
		t.Fatalf("Expected 4 comments, got %d", initialCommentCount)
	}

	t.Logf("Attempting to delete post with ID: %s", post1.ID)
	err := r.DeletePost(ctx, post1.ID)
	if err != nil {
		t.Fatalf("DeletePost failed: %v", err)
	}
	t.Logf("Post %s deleted successfully.", post1.ID)

	// Проверяем, что пост удален
	gotPost, _ := r.GetPostByID(ctx, post1.ID)
	if gotPost != nil {
		t.Error("Post should be deleted")
	} else {
		t.Log("Post successfully verified as deleted.")
	}

	// Проверяем, что комментарии поста удалились вместе с ним
	t.Log("Verifying associated comments are deleted...")
	checkCommentDeleted(t, r, c1.ID, "C1")
	checkCommentDeleted(t, r, c1_1.ID, "C1.1")
	checkCommentDeleted(t, r, c1_1_1.ID, "C1.1.1")

	// C1.2 был создан под post1, так что тоже должен быть удален.
	checkCommentDeleted(t, r, c1_2.ID, "C1.2")

	if len(r.comments) != 0 {
		t.Errorf("Expected 0 comments after deleting post and all its comments, got %d", len(r.comments))
	} else {
		t.Log("All comments associated with the post verified as deleted.")
	}

	// Удаление несуществующего поста
	nonExistentPostID := "non-existent-id"
	t.Logf("Attempting to delete non-existent post ID: %s", nonExistentPostID)
	err = r.DeletePost(ctx, nonExistentPostID)
	if err == nil {
		t.Error("Expected error for non-existent post, got nil")
	} else {
		t.Logf("Correctly received expected error for non-existent post: %v", err)
	}
}

func TestInMemoryRepositoryCreateComment(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()
	post1 := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	post2 := createTestPost(t, r, "user2", "Post 2", "Content 2", false)
	t.Logf("Created post1 (ID: %s, CommentsEnabled: %v), post2 (ID: %s, CommentsEnabled: %v)", post1.ID, post1.CommentsEnabled, post2.ID, post2.CommentsEnabled)

	// Успешное создание
	comment1 := &domain.Comment{
		ID:       uuid.New().String(),
		PostID:   post1.ID,
		AuthorID: "user1",
		Content:  "First comment",
	}
	t.Logf("Attempting to create comment for post1 (enabled)")
	err := r.CreateComment(ctx, comment1)
	if err != nil {
		t.Errorf("CreateComment failed: %v", err)
	}
	t.Logf("Successfully created comment with ID: %s", comment1.ID)

	// Ответ на комментарий
	comment1_1 := &domain.Comment{
		ID:       uuid.New().String(),
		PostID:   post1.ID,
		AuthorID: "user2",
		Content:  "Reply to first comment",
		ParentID: &comment1.ID,
	}
	t.Logf("Attempting to create reply for comment %s (PostID: %s)", comment1.ID, post1.ID)
	err = r.CreateComment(ctx, comment1_1)
	if err != nil {
		t.Errorf("CreateComment reply failed: %v", err)
	} else {
		t.Logf("Successfully created reply with ID: %s", comment1_1.ID)
	}

	// Комментарии отключены
	commentDisabled := &domain.Comment{
		PostID:   post2.ID,
		AuthorID: "user3",
		Content:  "Comment for disabled post",
	}
	t.Logf("Attempting to create comment for post2 (disabled)")
	err = r.CreateComment(ctx, commentDisabled)
	expectedErr := fmt.Sprintf("post with ID %s does not allow comments", post2.ID)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Expected error '%s' for disabled comments, got %v", expectedErr, err)
	} else {
		t.Logf("Correctly received expected error for disabled comments: %v", err)
	}

	// Несуществующий пост
	nonExistentPostID := "non-existent"
	commentNonExistentPost := &domain.Comment{
		PostID:   nonExistentPostID,
		AuthorID: "user4",
		Content:  "Comment for non-existent post",
	}
	t.Logf("Attempting to create comment for non-existent post ID: %s", nonExistentPostID)
	err = r.CreateComment(ctx, commentNonExistentPost)
	expectedErr = fmt.Sprintf("cannot create comment, post with ID %s does not exist", nonExistentPostID)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Expected error '%s' for non-existent post, got %v", expectedErr, err)
	} else {
		t.Logf("Correctly received expected error for non-existent post: %v", err)
	}

	// Ответ на несуществующий комментарий
	nonExistentParentID := "non-existent-parent"
	commentNonExistentParent := &domain.Comment{
		PostID:   post1.ID,
		AuthorID: "user5",
		Content:  "Reply to non-existent parent",
		ParentID: &nonExistentParentID,
	}
	t.Logf("Attempting to create reply to non-existent parent ID: %s", nonExistentParentID)
	err = r.CreateComment(ctx, commentNonExistentParent)
	expectedErr = fmt.Sprintf("cannot reply to nonexistent comment with ID %s", nonExistentParentID)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Expected error '%s' for non-existent parent, got %v", expectedErr, err)
	} else {
		t.Logf("Correctly received expected error for non-existent parent: %v", err)
	}
}

func TestInMemoryRepositoryGetCommentByID(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()
	post := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	comment := createTestComment(t, r, post.ID, "user1", "Comment 1", nil)
	t.Logf("Created comment with ID: %s", comment.ID)

	tests := []struct {
		name    string
		id      string
		want    *domain.Comment
		wantErr bool
	}{
		{"Found Comment", comment.ID, comment, false},
		{"Not Found", "non-existent-id", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Attempting to get comment by ID: %s", tt.id)
			got, err := r.GetCommentByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommentByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCommentByID() got = %v, want %v", got, tt.want)
			} else {
				if got != nil {
					t.Logf("Successfully retrieved comment with ID: %s", got.ID)
				} else {
					t.Log("Comment not found as expected.")
				}
			}
		})
	}
}

func checkCommentDeleted(t *testing.T, r *InMemoryRepository, commentID, name string) {
	gotComment, _ := r.GetCommentByID(context.Background(), commentID)
	if gotComment != nil {
		t.Errorf("Comment %s (ID: %s) should be deleted", name, commentID)
	} else {
		t.Logf("Comment %s (ID: %s) verified as deleted.", name, commentID)
	}
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

	t.Logf("Initial comments: C1(%s), C1.1(%s), C1.1.1(%s), C1.2(%s), C1.3(%s)", c1.ID, c1_1.ID, c1_1_1.ID, c1_2.ID, c1_3.ID)

	// Проверяем изначальное состояние
	initialCommentCount := len(r.comments)
	t.Logf("Initial total comment count in map: %d", initialCommentCount)
	if initialCommentCount != 5 {
		t.Fatalf("Expected 5 comments, got %d", initialCommentCount)
	}

	// Удаляем корневой комментарий C1 (должен удалить C1, C1.1, C1.1.1)
	t.Logf("Attempting to delete comment ID: %s (C1) and its branch...", c1.ID)
	err := r.DeleteComment(ctx, c1.ID)
	if err != nil {
		t.Fatalf("DeleteComment failed: %v", err)
	}
	t.Logf("Deletion of C1 branch (ID: %s) completed.", c1.ID)

	// Проверяем, что удалены 3 комментария
	if len(r.comments) != 2 { // Остались C1.2 и C1.3
		t.Errorf("Expected 2 comments after deleting C1 branch, got %d", len(r.comments))
	} else {
		t.Logf("Remaining comment count verified: %d", len(r.comments))
	}
	checkCommentDeleted(t, r, c1.ID, "C1")
	checkCommentDeleted(t, r, c1_1.ID, "C1.1")
	checkCommentDeleted(t, r, c1_1_1.ID, "C1.1.1")

	t.Log("Verifying C1.2 and C1.3 still exist...")
	if _, ok := r.comments[c1_2.ID]; !ok {
		t.Error("C1.2 should NOT be deleted")
	} else {
		t.Logf("C1.2 (ID: %s) still exists.", c1_2.ID)
	}
	if _, ok := r.comments[c1_3.ID]; !ok {
		t.Error("C1.3 should NOT be deleted")
	} else {
		t.Logf("C1.3 (ID: %s) still exists.", c1_3.ID)
	}

	// Проверяем, что индекс тоже корректен
	t.Log("Verifying commentsIndex state after deletion...")
	_, ok := r.commentsIndex[post1.ID][c1.ID] // c1 был родителем
	if ok {
		t.Error("C1 should be removed as a parent from index")
	} else {
		t.Logf("C1 (ID: %s) successfully removed as a parent from index.", c1.ID)
	}

	// C1.2 все еще корневой
	rootComments := r.commentsIndex[post1.ID][""]
	foundC1_2 := false
	for _, c := range rootComments {
		if c.ID == c1_2.ID {
			foundC1_2 = true
			break
		}
	}
	if !foundC1_2 {
		t.Error("C1.2 should still be in root comments index")
	} else {
		t.Logf("C1.2 (ID: %s) still present in root comments index.", c1_2.ID)
	}
	// C1.3 все еще корневой
	foundC1_3 := false
	for _, c := range rootComments {
		if c.ID == c1_3.ID {
			foundC1_3 = true
			break
		}
	}
	if !foundC1_3 {
		t.Error("C1.3 should still be in root comments index")
	} else {
		t.Logf("C1.3 (ID: %s) still present in root comments index.", c1_3.ID)
	}

	// Удаление несуществующего комментария
	nonExistentCommentID := "non-existent-id"
	t.Logf("Attempting to delete non-existent comment ID: %s", nonExistentCommentID)
	err = r.DeleteComment(ctx, nonExistentCommentID)
	if err == nil {
		t.Error("Expected error for non-existent comment, got nil")
	} else {
		t.Logf("Correctly received expected error for non-existent comment: %v", err)
	}
}

func TestInMemoryRepositorySetCommentsEnabled(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()
	post := createTestPost(t, r, "user1", "Post 1", "Content 1", true)
	t.Logf("Initial post ID: %s, CommentsEnabled: %v", post.ID, post.CommentsEnabled)

	// Выключаем комменты
	t.Logf("Attempting to disable comments for post ID: %s", post.ID)
	updatedPost, err := r.SetCommentsEnabled(ctx, post.ID, false)
	if err != nil {
		t.Fatalf("SetCommentsEnabled failed: %v", err)
	}
	if updatedPost.CommentsEnabled != false {
		t.Errorf("Expected comments to be disabled, got %v", updatedPost.CommentsEnabled)
	} else {
		t.Logf("Comments successfully disabled. Updated Post CommentsEnabled: %v", updatedPost.CommentsEnabled)
	}

	// Включаем комменты
	t.Logf("Attempting to enable comments for post ID: %s", post.ID)
	updatedPost, err = r.SetCommentsEnabled(ctx, post.ID, true)
	if err != nil {
		t.Fatalf("SetCommentsEnabled failed: %v", err)
	}
	if updatedPost.CommentsEnabled != true {
		t.Errorf("Expected comments to be enabled, got %v", updatedPost.CommentsEnabled)
	} else {
		t.Logf("Comments successfully enabled. Updated Post CommentsEnabled: %v", updatedPost.CommentsEnabled)
	}

	// Пытаемся переключить комменты на несуществующем посте
	nonExistentID := "non-existent-id"
	t.Logf("Attempting to set comments enabled for non-existent post ID: %s", nonExistentID)
	_, err = r.SetCommentsEnabled(ctx, nonExistentID, true)
	if err == nil {
		t.Error("Expected error for non-existent post, got nil")
	} else {
		t.Logf("Correctly received expected error for non-existent post: %v", err)
	}
}

func TestInMemoryRepositoryGetComments(t *testing.T) {
	r := NewInMemoryRepository()
	ctx := context.Background()

	t.Log("Creating test post...")
	post := createTestPost(t, r, "user1", "Post", "Content", true)

	t.Log("Creating comments...")
	time.Sleep(1 * time.Millisecond)
	c1 := createTestComment(t, r, post.ID, "u1", "C1", nil)

	time.Sleep(1 * time.Millisecond)
	c2 := createTestComment(t, r, post.ID, "u2", "C2", nil)

	time.Sleep(1 * time.Millisecond)
	c3 := createTestComment(t, r, post.ID, "u3", "C3", nil)

	t.Log("Creating nested comments...")
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
			t.Logf("Getting comments: postID=%s, parentID=%v, limit=%d, offset=%d",
				tt.postID, tt.parentID, tt.limit, tt.offset)

			got, err := r.GetComments(ctx, tt.postID, tt.parentID, tt.limit, tt.offset, tt.order)

			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				t.Logf("Received expected error: %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(got) != len(tt.wantIDs) {
				t.Fatalf("Expected %d comments, got %d", len(tt.wantIDs), len(got))
			}

			for i := range got {
				if got[i].ID != tt.wantIDs[i] {
					t.Errorf("At index %d got ID %s, want %s", i, got[i].ID, tt.wantIDs[i])
				}
			}

			t.Logf("Successfully retrieved %d comments", len(got))
			for i, c := range got {
				t.Logf("  [%d] Comment ID: %s", i, c.ID)
			}
		})
	}
}
