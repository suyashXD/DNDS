FROM golang:1.24-alpine AS builder


WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o notification-service ./cmd/server/

# Use a Docker multi-stage build to create a lean production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/notification-service .
# Copy the GraphQL schema
COPY --from=builder /app/internal/graphql/schema/schema.graphql ./internal/graphql/schema/

# Expose ports
EXPOSE 50051
EXPOSE 8080

# Command to run
CMD ["./notification-service"]