# Build stage
FROM golang:latest AS builder

# Set working directory
WORKDIR /app

# Copy everything including docs/ (pre-generated locally)
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage - using distroless for minimal size and security
FROM gcr.io/distroless/static-debian11

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main /app/main

# Copy config files if needed (optional)
COPY --from=builder /app/config /app/config

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/main"]
