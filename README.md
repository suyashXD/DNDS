# DNDS

A service that handles real-time notifications for a social media platform when users create new posts.

## Features

- gRPC endpoint for receiving new post events
- Concurrent notification processing using Go routines and worker pools
- GraphQL API for retrieving user notifications
- Automatic retry with exponential backoff for failed notifications
- Metrics endpoint for monitoring system performance

## Architecture

The service consists of the following components:

1. **gRPC Service**: Receives new post events and queues notifications for followers.
2. **Notification Queue**: Processes notifications concurrently using a worker pool.
3. **GraphQL API**: Provides an endpoint to retrieve user notifications.
4. **In-Memory Store**: Stores user, post, and notification data.

## Technical Details

- **Language**: Go
- **API Protocols**: gRPC, GraphQL
- **Concurrency**: Goroutines, Channels, Worker Pool
- **Error Handling**: Retry logic with exponential backoff

## Setup Instructions

### Prerequisites

- Go 1.22+ 
- Protocol Buffer Compiler (protoc)
- Git

### Installation

1. Clone the repository
   ```bash
   git clone https://github.com/suyashXD/DNDS.git
   cd DNDS
   ```

2. Install dependencies
   ```bash
   go mod download
   ```

3. Generate protocol buffer code
   ```bash
   protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       internal/grpc/proto/notification.proto
   ```

4. Build the service
   ```bash
   go build -o notification-service.exe ./cmd/server/
   ```

5. Run the service
   ```bash
   ./notification-service.exe
   ```

### Docker

Alternatively, you can use Docker:

```bash
# Build the image
docker build -t notification-service .

# Run the container
docker run -p 50051:50051 -p 8080:8080 notification-service
```

## Usage

The service exposes the following endpoints:

### gRPC API (Port 50051)

The gRPC service implements the following RPC:

```protobuf
rpc PublishPost(Post) returns (NotificationResponse)
```

Example using a gRPC client:

```go
post := &proto.Post{
    AuthorId: "user1",
    Content: "Hello world!",
    CreatedAt: time.Now().Unix(),
}
response, err := client.PublishPost(context.Background(), post)
```

### GraphQL API (Port 8080)

The GraphQL API is available at `http://localhost:8080/graphql` and provides the following query:

```graphql
query {
  getNotifications(userId: "user1") {
    id
    content
    createdAt
    status
  }
}
```

### Metrics API

Metrics are available at `http://localhost:8080/metrics` and return JSON with the following information:

- `total_sent`: Total number of notifications delivered
- `failed_attempts`: Number of failed delivery attempts
- `total_retries`: Number of retried deliveries
- `avg_delivery_time`: Average time to deliver a notification
- `queue_size`: Current number of notifications in the queue
- `worker_count`: Number of active workers

## Assumptions

- All data is stored in memory for simplicity
- Sample user and follower data is pre-populated
- A 10% random failure rate is simulated for demonstration purposes
- Notifications are kept simple with minimal content

## Future Improvements

- Replace in-memory storage with a persistent database
- Add authentication/authorization
- Implement rate limiting
- Scale horizontally with multiple instances