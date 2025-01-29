# Use the official Go image as a base
FROM golang:1.24-rc-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o build/posthoot cmd/main.go

# Build helper binary
RUN CGO_ENABLED=0 GOOS=linux go build -o build/helper cmd/helper/main.go

# Use a minimal alpine image for the final stage
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/build/posthoot .
COPY --from=builder /app/build/helper .

# Expose ports
EXPOSE 8080

# Set the entry point
ENTRYPOINT ["./posthoot"]

