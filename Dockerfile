# Build stage
FROM golang:1.22-alpine AS build

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /nexus-mind-vector-store ./vectorstore

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy the binary from the build stage
COPY --from=build /nexus-mind-vector-store /usr/local/bin/

# Create a non-root user to run the application
RUN adduser -D -h /home/appuser appuser
USER appuser
WORKDIR /home/appuser

# Create a data directory
RUN mkdir -p /home/appuser/data

# Set environment variables (can be overridden at runtime)
ENV NODE_ID="node-1" \
    HTTP_PORT=8080 \
    DIMENSIONS=128 \
    DISTANCE_FUNCTION="cosine" \
    LOG_LEVEL="info"

# Expose HTTP port
EXPOSE 8080

# Copy the entrypoint script
COPY --from=build /app/docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Set the entrypoint
ENTRYPOINT ["docker-entrypoint.sh"]