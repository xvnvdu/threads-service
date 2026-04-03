package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xvnvdu/threads-service/internal/domain"
	"github.com/xvnvdu/threads-service/internal/repository/mocks"
	"go.uber.org/mock/gomock"
)

func TestServiceSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	pubSub := NewCommentPubSub()
	s := NewService(mockRepo, pubSub)

	ctx := context.Background()
	postID := "post-for-sub"

	mockRepo.EXPECT().
		CreateComment(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	subChan, err := s.Subscribe(postID)
	require.NoError(t, err)
	require.NotNil(t, subChan)

	testComment := &domain.Comment{
		ID:       "comment-sub-1",
		PostID:   postID,
		AuthorID: "sub-tester",
		Content:  "A comment for subscription",
	}

	go func() {
		_, _ = s.CreateComment(ctx, testComment)
	}()

	select {
	case received := <-subChan:
		require.NotNil(t, received)

		assert.Equal(t, testComment.ID, received.ID)
		assert.Equal(t, testComment.Content, received.Content)

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for subscription event")
	}
}
