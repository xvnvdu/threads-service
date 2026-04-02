package service

import (
	"fmt"
	"log"
	"sync"

	"github.com/xvnvdu/threads-service/internal/domain"
)

// CommentPubSub - простая in-memory pub/sub система для комментариев,
// хранит мапу каналов, где ключ - это postID
type CommentPubSub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan *domain.Comment
}

func NewCommentPubSub() *CommentPubSub {
	return &CommentPubSub{
		subscribers: make(map[string][]chan *domain.Comment),
	}
}

// Subscribe подписывает клиента на новые комментарии для указанного postID,
// возвращает канал, через который будут приходить комментарии
func (ps *CommentPubSub) Subscribe(postID string) (<-chan *domain.Comment, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if postID == "" {
		return nil, fmt.Errorf("pubsub: postID cannot be empty for subscription")
	}

	eventChan := make(chan *domain.Comment, 1)

	ps.subscribers[postID] = append(ps.subscribers[postID], eventChan)
	log.Println("[INFO] pubsub: subscribed to post with id:", postID)

	return eventChan, nil
}

// Publish отправляет комментарий всем подписчикам данного postID
func (ps *CommentPubSub) Publish(postID string, comment *domain.Comment) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	subscribersForPost := make([]chan *domain.Comment, len(ps.subscribers[postID]))
	copy(subscribersForPost, ps.subscribers[postID])

	for _, subChan := range subscribersForPost {
		select {
		case subChan <- comment:
		default:
			// Подписчик не готов принять - канал заполнен или закрыт
			log.Printf(
				"[WARN] pubsub: subscriber for post %s (channel %p) not ready for comment %s\n",
				postID, subChan, comment.ID,
			)
		}
	}
	log.Println("[INFO] pubsub: published notification to all subscribers to post with id:", postID)
}
