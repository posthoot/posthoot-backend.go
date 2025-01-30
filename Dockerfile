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
FROM gcr.io/distroless/static-debian12

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/build/posthoot .
COPY --from=builder /app/build/helper .
COPY --from=builder /app/docker-entrypoint.sh .
# Copy template seeder data for Airley templates
# Source: /app/internal/models/seeder/airley/templates.json
# Destination: /app/internal/models/seeder/airley/templates.json
COPY --from=builder /app/internal/models/seeder/airley/templates.json /app/internal/models/seeder/airley/

# Copy all initial setup seeder files for database initialization 
# Source: /app/internal/models/seeder/initial-setup/*
# Destination: /app/internal/models/seeder/initial-setup/
COPY --from=builder /app/internal/models/seeder/initial-setup/* /app/internal/models/seeder/initial-setup/
RUN chmod +x docker-entrypoint.sh

# Install required packages
RUN apk add --no-cache curl netcat-openbsd

# Expose ports
EXPOSE 8080

# Set the entry point
ENTRYPOINT ["./docker-entrypoint.sh"]
