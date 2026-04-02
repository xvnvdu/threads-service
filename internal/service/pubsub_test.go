package service

import (
	"context"
	"testing"
	"time"

	"github.com/xvnvdu/threads-service/internal/domain"
	"github.com/xvnvdu/threads-service/internal/repository/mocks"
	"go.uber.org/mock/gomock"
)

func TestService_Subscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()
	postID := "post-for-sub"
	authorID := "sub-tester"

	t.Logf("subscribing to comments for post ID: %s", postID)
	subChan, err := s.Subscribe(postID)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	testComment := &domain.Comment{
		ID:       "comment-sub-1",
		PostID:   postID,
		AuthorID: authorID,
		Content:  "A comment for subscription",
	}

	go func() {
		mockRepo.EXPECT().CreateComment(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		t.Logf("simulating CreateComment for post ID: %s", postID)
		_, err := s.CreateComment(ctx, testComment)
		if err != nil {
			t.Logf("error in CreateComment goroutine: %v", err)
		}
	}()

	t.Log("waiting for comment on subscription channel...")
	select {
	case receivedComment := <-subChan:
		t.Logf("successfully received comment with ID: %s", receivedComment.ID)
		if receivedComment.ID != testComment.ID {
			t.Errorf("received comment ID %s, expected %s", receivedComment.ID, testComment.ID)
		}
		if receivedComment.Content != testComment.Content {
			t.Errorf("received comment content '%s', expected '%s'", receivedComment.Content,
				testComment.Content)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for comment on subscription channel")
	}
}
