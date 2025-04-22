package queue

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/suyashXD/DNDS/internal/models"
	"github.com/suyashXD/DNDS/internal/store"
)

const (
	maxRetries       = 3
	initialBackoff   = 100 * time.Millisecond
	failureRate      = 0.1 // 10% failure rate
	maxWorkers       = 10  // Maximum number of concurrent workers
)

// NotificationQueue handles the queuing and processing of notifications
type NotificationQueue struct {
	store        *store.MemoryStore
	queue        chan *models.Notification
	wg           sync.WaitGroup
	workerCount  int
	metrics      *Metrics
	ctx          context.Context
	cancel       context.CancelFunc
}

// Metrics tracks statistics about notification deliveries
type Metrics struct {
	TotalSent      int64
	FailedAttempts int64
	TotalRetries   int64
	mu             sync.RWMutex
	deliveryTimes  []time.Duration
}

// NewNotificationQueue creates a new notification queue with the specified store
func NewNotificationQueue(store *store.MemoryStore, workerCount int) *NotificationQueue {
	if workerCount <= 0 {
		workerCount = maxWorkers
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	return &NotificationQueue{
		store:       store,
		queue:       make(chan *models.Notification, 1000), // Buffer size of 1000
		workerCount: workerCount,
		metrics: &Metrics{
			deliveryTimes: make([]time.Duration, 0),
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins processing notifications with the worker pool
func (nq *NotificationQueue) Start() {
	for i := 0; i < nq.workerCount; i++ {
		nq.wg.Add(1)
		go nq.worker(i)
	}
	log.Printf("Started notification queue with %d workers", nq.workerCount)
}

// Stop gracefully shuts down the queue
func (nq *NotificationQueue) Stop() {
	nq.cancel()
	nq.wg.Wait()
	close(nq.queue)
	log.Println("Notification queue stopped")
}

// QueueNotification adds a notification to the processing queue
func (nq *NotificationQueue) QueueNotification(notification *models.Notification) {
	select {
	case nq.queue <- notification:
		// Successfully queued
	default:
		// Queue is full, handle gracefully
		log.Printf("Queue is full, notification %s dropped", notification.ID)
	}
}

// QueueNotifications adds multiple notifications to the queue
func (nq *NotificationQueue) QueueNotifications(notifications []*models.Notification) int {
	queued := 0
	for _, notification := range notifications {
		select {
		case nq.queue <- notification:
			queued++
		default:
			// Queue is full
			log.Printf("Queue is full, notification %s dropped", notification.ID)
		}
	}
	return queued
}

// worker processes notifications from the queue
func (nq *NotificationQueue) worker(id int) {
	defer nq.wg.Done()
	
	log.Printf("Worker %d started", id)
	
	for {
		select {
		case notification, ok := <-nq.queue:
			if !ok {
				log.Printf("Worker %d shutting down", id)
				return
			}
			nq.processNotification(notification)
		case <-nq.ctx.Done():
			log.Printf("Worker %d received shutdown signal", id)
			return
		}
	}
}

// processNotification handles the delivery of a notification with retry logic
func (nq *NotificationQueue) processNotification(notification *models.Notification) {
	startTime := time.Now()
	
	// Simulate processing delay (10-50ms)
	time.Sleep(time.Duration(10+rand.Intn(40)) * time.Millisecond)
	
	// Simulate random failures (10% chance)
	if rand.Float64() < failureRate {
		nq.metrics.mu.Lock()
		nq.metrics.FailedAttempts++
		nq.metrics.mu.Unlock()
		
		notification.Attempts++
		
		if notification.Attempts <= maxRetries {
			// Calculate exponential backoff
			backoff := time.Duration(math.Pow(2, float64(notification.Attempts-1))) * initialBackoff
			
			log.Printf("Notification %s to user %s failed (attempt %d/%d), retrying in %v",
				notification.ID, notification.UserID, notification.Attempts, maxRetries, backoff)
			
			notification.Status = models.StatusRetrying
			err := nq.store.UpdateNotification(notification)
			if err != nil {
				log.Printf("Failed to update notification status: %v", err)
			}
			
			nq.metrics.mu.Lock()
			nq.metrics.TotalRetries++
			nq.metrics.mu.Unlock()
			
			// Schedule retry after backoff
			go func(n *models.Notification, d time.Duration) {
				time.Sleep(d)
				nq.QueueNotification(n)
			}(notification, backoff)
			
			return
		} else {
			// Max retries reached
			log.Printf("Notification %s to user %s failed permanently after %d attempts", 
				notification.ID, notification.UserID, notification.Attempts)
			
			notification.Status = models.StatusFailed
			err := nq.store.UpdateNotification(notification)
			if err != nil {
				log.Printf("Failed to update notification status: %v", err)
			}
			
			return
		}
	}
	
	// Successful delivery
	notification.Status = models.StatusDelivered
	err := nq.store.UpdateNotification(notification)
	if err != nil {
		log.Printf("Failed to update notification status: %v", err)
	}
	
	// Record metrics
	deliveryTime := time.Since(startTime)
	nq.metrics.mu.Lock()
	nq.metrics.TotalSent++
	nq.metrics.deliveryTimes = append(nq.metrics.deliveryTimes, deliveryTime)
	nq.metrics.mu.Unlock()
	
	fmt.Printf("Notification sent to User%s for Post%s\n", notification.UserID, notification.PostID)
}

// GetMetrics returns the current metrics
func (nq *NotificationQueue) GetMetrics() map[string]interface{} {
	nq.metrics.mu.RLock()
	defer nq.metrics.mu.RUnlock()
	
	var avgDeliveryTime time.Duration
	if len(nq.metrics.deliveryTimes) > 0 {
		var sum time.Duration
		for _, t := range nq.metrics.deliveryTimes {
			sum += t
		}
		avgDeliveryTime = sum / time.Duration(len(nq.metrics.deliveryTimes))
	}
	
	return map[string]interface{}{
		"total_sent":        nq.metrics.TotalSent,
		"failed_attempts":   nq.metrics.FailedAttempts,
		"total_retries":     nq.metrics.TotalRetries,
		"avg_delivery_time": avgDeliveryTime.String(),
		"queue_size":        len(nq.queue),
		"worker_count":      nq.workerCount,
	}
}