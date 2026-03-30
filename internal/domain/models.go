package domain

import "time"

// Post представляет собой модель поста в системе
type Post struct {
	ID              string
	Title           string
	Content         string
	AuthorID        string
	CommentsEnabled bool
	CreatedAt       time.Time
	Comments        []*Comment
}

// Comment представляет собой модель комментария к посту
type Comment struct {
	ID        string
	PostID    string
	ParentID  *string
	Content   string
	AuthorID  string
	CreatedAt time.Time
	Children  []*Comment
}
