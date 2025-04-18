# Stage 1: Build the application
FROM golang:1.22 AS builder

WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gemfactory main.go

# Stage 2: Create the final image
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/gemfactory .

# Copy .env file (will be overridden by Docker Compose or runtime environment)
COPY .env .

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Run the application
CMD ["./gemfactory"]