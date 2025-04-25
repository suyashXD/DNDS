package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"google.golang.org/grpc"

	"github.com/suyashXD/DNDS/internal/grpc/proto"
	"github.com/suyashXD/DNDS/internal/grpc/service"
	"github.com/suyashXD/DNDS/internal/graphql/resolver"
	"github.com/suyashXD/DNDS/internal/queue"
	"github.com/suyashXD/DNDS/internal/store"
)

const (
	grpcPort        = 50051
	httpPort        = 8080
	workerCount     = 5
	shutdownTimeout = 10 * time.Second
)

func main() {
	// Create store with sample data
	memoryStore := store.NewMemoryStore(true)
	
	// Create notification queue
	notificationQueue := queue.NewNotificationQueue(memoryStore, workerCount)
	notificationQueue.Start()
	
	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Create gRPC server
	go serveGRPC(ctx, memoryStore, notificationQueue)
	
	// Create HTTP/GraphQL server
	go serveHTTP(ctx, memoryStore, notificationQueue)
	
	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down servers...")
	cancel()
	
	// Shutdown notification queue
	notificationQueue.Stop()
	
	log.Println("Server gracefully stopped")
}

func serveGRPC(ctx context.Context, store *store.MemoryStore, queue *queue.NotificationQueue) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", grpcPort, err)
	}
	
	notificationService := service.NewNotificationService(store, queue)
	
	grpcServer := grpc.NewServer()
	proto.RegisterNotificationServiceServer(grpcServer, notificationService)
	
	log.Printf("gRPC server started on port %d", grpcPort)
	
	go func() {
		<-ctx.Done()
		log.Println("Stopping gRPC server...")
		grpcServer.GracefulStop()
	}()
	
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func serveHTTP(ctx context.Context, store *store.MemoryStore, queue *queue.NotificationQueue) {
	// Load GraphQL schema
	schemaContent, err := ioutil.ReadFile("internal/graphql/schema/schema.graphql")
	if err != nil {
		log.Fatalf("Failed to read schema: %v", err)
	}
	
	// Create resolver
	r := resolver.NewResolver(store, queue)
	
	// Parse schema
	schema := graphql.MustParseSchema(string(schemaContent), r)
	
	// Create GraphQL handler
	graphqlHandler := &relay.Handler{Schema: schema}
	
	// Setup HTTP server
	mux := http.NewServeMux()
	
	// GraphQL endpoint
	mux.Handle("/graphql", graphqlHandler)
	
	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := queue.GetMetrics()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})
	
	// Simple health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Create server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: mux,
	}
	
	// Start server
	log.Printf("HTTP server started on port %d", httpPort)
	log.Printf("GraphQL endpoint available at http://localhost:%d/graphql", httpPort)
	log.Printf("Metrics available at http://localhost:%d/metrics", httpPort)
	
	go func() {
		<-ctx.Done()
		log.Println("Stopping HTTP server...")
		
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("HTTP server shutdown error: %v", err)
		}
	}()
	
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}