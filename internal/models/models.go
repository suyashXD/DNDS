package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	FollowerIDs []string `json:"follower_ids"`
	FollowingIDs []string `json:"following_ids"`
}

// Post represents a user's social media post
type Post struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationStatus represents the current status of a notification
type NotificationStatus int

const (
	StatusUnknown NotificationStatus = iota
	StatusQueued
	StatusDelivered
	StatusFailed
	StatusRetrying
)

// Notification represents a single notification for a user
type Notification struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	PostID    string            `json:"post_id"`
	AuthorID  string            `json:"author_id"`
	Content   string            `json:"content"`
	CreatedAt time.Time         `json:"created_at"`
	Read      bool              `json:"read"`
	Status    NotificationStatus `json:"status"`
	Attempts  int               `json:"attempts"`
}

// NewNotification creates a new notification for a user about a post
func NewNotification(userID string, post *Post) *Notification {
	return &Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		PostID:    post.ID,
		AuthorID:  post.AuthorID,
		Content:   "New post from a user you follow",
		CreatedAt: time.Now(),
		Read:      false,
		Status:    StatusQueued,
		Attempts:  0,
	}
}