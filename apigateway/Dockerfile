# Build stage
FROM golang:1.17-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o apigateway main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy binary and entrypoint script from builder
COPY --from=builder /app/apigateway .
COPY --from=builder /app/scripts/entrypoint.sh /app/scripts/

# Make entrypoint script executable
RUN chmod +x /app/scripts/entrypoint.sh

# Expose port
EXPOSE 8090

# Set default environment
ENV APP_ENV=development

# Run the application
ENTRYPOINT ["/app/scripts/entrypoint.sh"]
CMD ["./apigateway"]
