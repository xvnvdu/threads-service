package inmemory

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/xvnvdu/threads-service/internal/domain"
	repo "github.com/xvnvdu/threads-service/internal/repository"
)

// InMemoryRepository реализует интерфейс
// Repository, используя in-memory хранилище
type InMemoryRepository struct {
	mu            sync.RWMutex
	posts         map[string]*domain.Post                 // или "", если коммент верхнего уровня
	comments      map[string]*domain.Comment              //                   v
	commentsIndex map[string]map[string][]*domain.Comment // map[postID]map[parentID][children]
}

// NewInMemoryRepository инициирует
// и возвращает новое хранилище
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		posts:         make(map[string]*domain.Post),
		comments:      make(map[string]*domain.Comment),
		commentsIndex: make(map[string]map[string][]*domain.Comment),
	}
}

// Здесь мы проверяем, что InMemoryRepository
// корректно реализует интерфейс Repository
var _ repo.Repository = (*InMemoryRepository)(nil)

// CreatePost сохраняет пост в in-memory хранилище
func (r *InMemoryRepository) CreatePost(ctx context.Context, post *domain.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.posts[post.ID] = post
	log.Println("[INFO] successfully created and saved post to in-memory storage:", post.ID)
	return nil
}

// GetPostByID возвращает пост из in-memory хранилища, если таковой имеется
func (r *InMemoryRepository) GetPostByID(ctx context.Context, id string) (*domain.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	post, ok := r.posts[id]
	if !ok {
		log.Println("[WARN] could not find post in in-memory storage:", post.ID)
		return nil, nil
	}
	log.Println("[INFO] successfully retrieved post from in-memory storage:", post.ID)
	return post, nil
}

// GetPosts возвращает limit постов с указанным offset
func (r *InMemoryRepository) GetPosts(ctx context.Context, limit, offset int) ([]*domain.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	postsList := make([]*domain.Post, 0, len(r.posts))
	for _, post := range r.posts {
		postsList = append(postsList, post)
	}

	if offset >= len(postsList) {
		return []*domain.Post{}, nil
	}

	// Сортировка по новизне создания
	sort.SliceStable(postsList, func(i, j int) bool {
		return postsList[i].CreatedAt.After(postsList[j].CreatedAt)
	})

	end := min(offset+limit, len(postsList))
	log.Println("[INFO] successfully retrieved posts from in-memory storage")
	return postsList[offset:end], nil
}

// SetCommentsEnabled переключает возможность оставлять комментарии под постом
func (r *InMemoryRepository) SetCommentsEnabled(ctx context.Context, postID string, enabled bool) (*domain.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	post, ok := r.posts[postID]
	if !ok {
		return nil, fmt.Errorf("post with ID %s not found", postID)
	}
	post.CommentsEnabled = enabled
	log.Println("[INFO] successfully set CommentsEnabled:", enabled)
	return post, nil
}

// DeletePost удаляет пост и все связанные с ним комментарии
func (r *InMemoryRepository) DeletePost(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.posts[id]; !ok {
		return fmt.Errorf("post with ID %s not found, failed to delete", id)
	}

	if postIndex, ok := r.commentsIndex[id]; ok {
		rootComments := postIndex[""]

		ids := make([]string, 0, len(rootComments))
		for _, c := range rootComments {
			ids = append(ids, c.ID)
		}

		for _, id := range ids {
			r.deleteCommentRecursively(ctx, id)
		}
	}

	delete(r.commentsIndex, id)
	delete(r.posts, id)
	log.Println("[INFO] successfully deleted post from in-memory storage:", id)
	return nil
}

// CreateComment сохраняет комментарий в in-memory хранилище
func (r *InMemoryRepository) CreateComment(ctx context.Context, comment *domain.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	post, ok := r.posts[comment.PostID]
	if !ok {
		return fmt.Errorf("cannot create comment, post with ID %s does not exist", comment.PostID)
	}
	if !post.CommentsEnabled {
		return fmt.Errorf("post with ID %s does not allow comments", post.ID)
	}

	parentID := ""
	if comment.ParentID != nil {
		if _, ok := r.comments[*comment.ParentID]; !ok {
			return fmt.Errorf("cannot reply to nonexistent comment with ID %s", *comment.ParentID)
		}
		parentID = *comment.ParentID
	}

	// Проверяем в индексе, существует ли уже мапа с комментами для этого поста
	if _, ok := r.commentsIndex[post.ID]; !ok {
		r.commentsIndex[post.ID] = make(map[string][]*domain.Comment)
	}
	// Добавляем коммент в индекс
	r.commentsIndex[post.ID][parentID] = append(r.commentsIndex[post.ID][parentID], comment)
	// И добавляем в общее хранилище
	r.comments[comment.ID] = comment

	log.Println("[INFO] successfully created and saved comment to in-memory storage:", comment.ID)
	return nil
}

// GetCommentByID возвращает комментарий из in-memory хранилища, если таковой имеется
func (r *InMemoryRepository) GetCommentByID(ctx context.Context, id string) (*domain.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comment, ok := r.comments[id]
	if !ok {
		log.Println("[WARN] could not find comment in in-memory storage:", id)
		return nil, nil
	}
	log.Println("[INFO] successfully retrieved comment from in-memory storage:", id)
	return comment, nil
}

// GetComments возвращает limit комментариев под постом с указанными offset и родителем
func (r *InMemoryRepository) GetComments(ctx context.Context, postID string, parentID *string, limit, offset int, order repo.CommentSortOrder) ([]*domain.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.posts[postID]; !ok {
		return nil, fmt.Errorf("cannot get comments under nonexistent post with ID %s", postID)
	}

	parentKey := ""
	if parentID != nil {
		parentKey = *parentID
	}

	// Благодаря индексу получаем список нужных комментов за O(1)
	original := r.commentsIndex[postID][parentKey]
	commentsList := make([]*domain.Comment, len(original))
	copy(commentsList, original)

	if offset >= len(commentsList) {
		return []*domain.Comment{}, nil
	}

	sort.SliceStable(commentsList, func(i, j int) bool {
		// Комментарии верхнего уровня по умолчанию сортируются
		// от новых к старым, то есть сервис проверяет parentID == nil
		// и на его основе определяет порядок. В целом, конечно, можно
		// сделать так, чтобы клиент сам выбирал порядок комментов
		// верхнего уровня, тогда нужно будет менять схему
		if order == repo.CommentSortNewestFirst {
			return commentsList[i].CreatedAt.After(commentsList[j].CreatedAt)
		}
		// Ответы на комментарии сортируются по хронологии
		return commentsList[i].CreatedAt.Before(commentsList[j].CreatedAt)
	})

	end := min(offset+limit, len(commentsList))
	log.Println("[INFO] successfully retrieved comments from in-memory storage")
	return commentsList[offset:end], nil
}

// DeleteComment является оберткой над приватной рекурсивной функцией удаления комментариев
func (r *InMemoryRepository) DeleteComment(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.comments[id]
	if !ok {
		return fmt.Errorf("comment with ID %s not found, failed to delete", id)
	}

	r.deleteCommentRecursively(ctx, id)
	log.Println("[INFO] successfully deleted comment from in-memory storage:", id)
	return nil
}

// DeleteComment удаляет корневой комментарий и рекурсивно все его дочерние
func (r *InMemoryRepository) deleteCommentRecursively(ctx context.Context, commentID string) {
	comment, ok := r.comments[commentID]
	if !ok {
		return
	}

	// Получаем ID поста и всех детей текущего коммента
	postID := comment.PostID
	children := r.commentsIndex[postID][commentID]

	childrenCopy := make([]*domain.Comment, len(children))
	copy(childrenCopy, children)

	// Уходим в рекурсию по детям
	for _, child := range childrenCopy {
		r.deleteCommentRecursively(ctx, child.ID)
	}

	parentKey := ""
	if comment.ParentID != nil {
		parentKey = *comment.ParentID
	}

	// Удаляем комментарий на текущем уровне среди его родственников:
	// сдвигаем слайс, зануляем и удаляем последний элемент чтобы
	// избежать memory leak и GC спокойно избавился от указателя
	siblings := r.commentsIndex[postID][parentKey]
	for i, c := range siblings {
		if c.ID == commentID {
			copy(siblings[i:], siblings[i+1:])
			siblings[len(siblings)-1] = nil
			siblings = siblings[:len(siblings)-1]
			break
		}
	}
	r.commentsIndex[postID][parentKey] = siblings

	delete(r.commentsIndex[postID], commentID)
	delete(r.comments, commentID)
}
