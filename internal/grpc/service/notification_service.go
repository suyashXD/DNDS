package service

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/suyashXD/DNDS/internal/grpc/proto"
	"github.com/suyashXD/DNDS/internal/models"
	"github.com/suyashXD/DNDS/internal/queue"
	"github.com/suyashXD/DNDS/internal/store"
)

// NotificationService implements the gRPC NotificationService
type NotificationService struct {
	proto.UnimplementedNotificationServiceServer
	store *store.MemoryStore
	queue *queue.NotificationQueue
}

// NewNotificationService creates a new notification service
func NewNotificationService(store *store.MemoryStore, queue *queue.NotificationQueue) *NotificationService {
	return &NotificationService{
		store: store,
		queue: queue,
	}
}

// PublishPost handles new post events and triggers notifications to followers
func (s *NotificationService) PublishPost(ctx context.Context, req *proto.Post) (*proto.NotificationResponse, error) {
	// Create an internal post model from the request
	post := &models.Post{
		ID:        req.Id,
		AuthorID:  req.AuthorId,
		Content:   req.Content,
		CreatedAt: time.Unix(req.CreatedAt, 0),
	}

	// If post ID is empty, generate one
	if post.ID == "" {
		post.ID = uuid.New().String()
	}

	// If created_at is zero, set it to now
	if post.CreatedAt.IsZero() {
		post.CreatedAt = time.Now()
	}

	// Save the post
	err := s.store.SavePost(post)
	if err != nil {
		log.Printf("Failed to save post: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to save post: %v", err)
	}

	// Get the author's followers
	followers, err := s.store.GetFollowers(post.AuthorID)
	if err != nil {
		log.Printf("Failed to get followers: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get followers: %v", err)
	}

	// Create notifications for each follower
	notifications := make([]*models.Notification, 0, len(followers))
	for _, follower := range followers {
		notification := models.NewNotification(follower.ID, post)
		
		// Save the notification first
		err := s.store.SaveNotification(notification)
		if err != nil {
			log.Printf("Failed to save notification for user %s: %v", follower.ID, err)
			continue
		}
		
		notifications = append(notifications, notification)
	}

	// Queue notifications for delivery
	queuedCount := s.queue.QueueNotifications(notifications)

	log.Printf("Post %s published by %s, queued %d notifications", post.ID, post.AuthorID, queuedCount)

	// Return response
	return &proto.NotificationResponse{
		PostId:             post.ID,
		NotificationsQueued: int32(queuedCount),
		Success:            true,
	}, nil
}