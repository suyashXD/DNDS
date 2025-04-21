package store

import (
	"errors"
	"sync"
	"time"

	"github.com/suyashXD/Distributed-Notification-Delivery-Service/internal/models"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrPostNotFound        = errors.New("post not found")
	ErrNotificationNotFound = errors.New("notification not found")
)

// MemoryStore implements an in-memory data store for the application

type MemoryStore struct {
	users         map[string]*models.User
	posts         map[string]*models.Post
	notifications map[string][]*models.Notification
	mu            sync.RWMutex
}

// NewMemoryStore creates a new in-memory store with optional sample data

func NewMemoryStore(loadSampleData bool) *MemoryStore {
	store := &MemoryStore{
		users:         make(map[string]*models.User),
		posts:         make(map[string]*models.Post),
		notifications: make(map[string][]*models.Notification),
	}

	if loadSampleData {
		store.loadSampleData()
	}

	return store
}

// GetUser retrieves a user by ID

func (s *MemoryStore) GetUser(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetAllUsers returns all users

func (s *MemoryStore) GetAllUsers() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*models.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

// GetFollowers returns all followers for a user

func (s *MemoryStore) GetFollowers(userID string) ([]*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}

	followers := make([]*models.User, 0, len(user.FollowerIDs))
	for _, id := range user.FollowerIDs {
		if follower, ok := s.users[id]; ok {
			followers = append(followers, follower)
		}
	}
	return followers, nil
}

// SavePost stores a new post

func (s *MemoryStore) SavePost(post *models.Post) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.posts[post.ID] = post
	return nil
}

// GetPost retrieves a post by ID
func (s *MemoryStore) GetPost(id string) (*models.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	post, exists := s.posts[id]
	if !exists {
		return nil, ErrPostNotFound
	}
	return post, nil
}

// SaveNotification adds a notification for a user
func (s *MemoryStore) SaveNotification(notification *models.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.notifications[notification.UserID] = append(s.notifications[notification.UserID], notification)
	return nil
}

// UpdateNotification updates a notification's status
func (s *MemoryStore) UpdateNotification(notification *models.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userNotifications, exists := s.notifications[notification.UserID]
	if !exists {
		return ErrUserNotFound
	}

	for i, n := range userNotifications {
		if n.ID == notification.ID {
			userNotifications[i] = notification
			return nil
		}
	}

	return ErrNotificationNotFound
}

// GetUserNotifications returns notifications for a user
func (s *MemoryStore) GetUserNotifications(userID string, limit int) ([]*models.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notifications, exists := s.notifications[userID]
	if !exists {
		return []*models.Notification{}, nil
	}

	// Return the most recent notifications up to the limit
	result := make([]*models.Notification, 0, limit)
	count := 0
	
	// Start from the end (most recent) and work backwards
	for i := len(notifications) - 1; i >= 0 && count < limit; i-- {
		result = append(result, notifications[i])
		count++
	}

	return result, nil
}

// loadSampleData populates the store with sample data
func (s *MemoryStore) loadSampleData() {
	// Create users
	users := []*models.User{
		{ID: "user1", Username: "alice", FollowerIDs: []string{}, FollowingIDs: []string{}},
		{ID: "user2", Username: "bob", FollowerIDs: []string{}, FollowingIDs: []string{}},
		{ID: "user3", Username: "charlie", FollowerIDs: []string{}, FollowingIDs: []string{}},
		{ID: "user4", Username: "dave", FollowerIDs: []string{}, FollowingIDs: []string{}},
		{ID: "user5", Username: "eve", FollowerIDs: []string{}, FollowingIDs: []string{}},
		{ID: "user6", Username: "frank", FollowerIDs: []string{}, FollowingIDs: []string{}},
		{ID: "user7", Username: "grace", FollowerIDs: []string{}, FollowingIDs: []string{}},
	}

	// Set up follower relationships
	// User1 (Alice) is followed by everyone
	users[0].FollowerIDs = []string{"user2", "user3", "user4", "user5", "user6", "user7"}
	
	// User2 (Bob) is followed by some users
	users[1].FollowerIDs = []string{"user1", "user3", "user5"}
	
	// User3 (Charlie) is followed by some users
	users[2].FollowerIDs = []string{"user1", "user2", "user4"}

	// User4 (Dave) is followed by some users
	users[3].FollowerIDs = []string{"user2", "user5", "user7"}

	// Map users to the store
	for _, user := range users {
		s.users[user.ID] = user
	}

	// Create some sample posts
	posts := []*models.Post{
		{
			ID:        "post1",
			AuthorID:  "user1",
			Content:   "Hello world from Alice!",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			ID:        "post2",
			AuthorID:  "user2",
			Content:   "Bob's first post",
			CreatedAt: time.Now().Add(-12 * time.Hour),
		},
		{
			ID:        "post3",
			AuthorID:  "user3",
			Content:   "Charlie's thoughts on distributed systems",
			CreatedAt: time.Now().Add(-6 * time.Hour),
		},
	}

	// Map posts to the store
	for _, post := range posts {
		s.posts[post.ID] = post
	}

	// Initialize empty notification lists for each user
	for _, user := range users {
		s.notifications[user.ID] = []*models.Notification{}
	}
}