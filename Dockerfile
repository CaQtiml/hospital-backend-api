# Stage 1: Build the Go application
FROM golang:1.24.2-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies.
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app - Creates a static binary
# CGO_ENABLED=0 is important for static linking with Alpine
# -ldflags="-w -s" strips debug symbols and reduces binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /hospital-middleware ./cmd/server/main.go

# Stage 2: Create the final image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /hospital-middleware /app/hospital-middleware

# Expose the port the Go application listens on (defined in .env and config.go)
# For Documentation, the actual port mapping happens in docker-compose.yml
EXPOSE 8080

# Run the application
CMD ["/app/hospital-middleware"]