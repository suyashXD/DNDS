package resolver

import (
	"context"
	"log"

	"github.com/graph-gophers/graphql-go"
	"github.com/suyashXD/DNDS/internal/models"
	"github.com/suyashXD/DNDS/internal/queue"
	"github.com/suyashXD/DNDS/internal/store"
)

// Resolver is the root resolver for GraphQL queries
type Resolver struct {
	store *store.MemoryStore
	queue *queue.NotificationQueue
}

// NewResolver creates a new GraphQL resolver
func NewResolver(store *store.MemoryStore, queue *queue.NotificationQueue) *Resolver {
	return &Resolver{
		store: store,
		queue: queue,
	}
}

// NotificationStatus represents the GraphQL enum for notification status
type NotificationStatus string

// NotificationStatusFromModel converts the model status to GraphQL enum
func NotificationStatusFromModel(status models.NotificationStatus) NotificationStatus {
	switch status {
	case models.StatusUnknown:
		return "UNKNOWN"
	case models.StatusQueued:
		return "QUEUED"
	case models.StatusDelivered:
		return "DELIVERED"
	case models.StatusFailed:
		return "FAILED"
	case models.StatusRetrying:
		return "RETRYING"
	default:
		return "UNKNOWN"
	}
}

// Notification resolver for GraphQL Notification type
type NotificationResolver struct {
	notification *models.Notification
}

func (r *NotificationResolver) ID() graphql.ID {
	return graphql.ID(r.notification.ID)
}

func (r *NotificationResolver) UserID() graphql.ID {
	return graphql.ID(r.notification.UserID)
}

func (r *NotificationResolver) PostID() graphql.ID {
	return graphql.ID(r.notification.PostID)
}

func (r *NotificationResolver) AuthorID() graphql.ID {
	return graphql.ID(r.notification.AuthorID)
}

func (r *NotificationResolver) Content() string {
	return r.notification.Content
}

func (r *NotificationResolver) CreatedAt() string {
	return r.notification.CreatedAt.Format("2006-01-02T15:04:05Z")
}

func (r *NotificationResolver) Read() bool {
	return r.notification.Read
}

func (r *NotificationResolver) Status() NotificationStatus {
	return NotificationStatusFromModel(r.notification.Status)
}

func (r *NotificationResolver) Attempts() int32 {
	return int32(r.notification.Attempts)
}

// MetricsResolver resolver for GraphQL Metrics type
type MetricsResolver struct {
	metrics map[string]interface{}
}

func (r *MetricsResolver) TotalSent() int32 {
	return int32(r.metrics["total_sent"].(int64))
}

func (r *MetricsResolver) FailedAttempts() int32 {
	return int32(r.metrics["failed_attempts"].(int64))
}

func (r *MetricsResolver) TotalRetries() int32 {
	return int32(r.metrics["total_retries"].(int64))
}

func (r *MetricsResolver) AvgDeliveryTime() string {
	return r.metrics["avg_delivery_time"].(string)
}

func (r *MetricsResolver) QueueSize() int32 {
	return int32(r.metrics["queue_size"].(int))
}

func (r *MetricsResolver) WorkerCount() int32 {
	return int32(r.metrics["worker_count"].(int))
}

// GetNotifications resolves the getNotifications query
func (r *Resolver) GetNotifications(ctx context.Context, args struct{ UserID graphql.ID }) ([]*NotificationResolver, error) {
	userID := string(args.UserID)
	
	// Get the latest 20 notifications for this user
	notifications, err := r.store.GetUserNotifications(userID, 20)
	if err != nil {
		log.Printf("Error retrieving notifications: %v", err)
		return nil, err
	}
	
	// Convert to resolvers
	resolvers := make([]*NotificationResolver, len(notifications))
	for i, notification := range notifications {
		resolvers[i] = &NotificationResolver{notification: notification}
	}
	
	return resolvers, nil
}

// GetMetrics resolves the getMetrics query
func (r *Resolver) GetMetrics(ctx context.Context) (*MetricsResolver, error) {
	metrics := r.queue.GetMetrics()
	return &MetricsResolver{metrics: metrics}, nil
}