# Use the official Golang image as a base image
FROM golang:1.25.2-alpine AS builder

# Set the current working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod .
COPY go.sum .

# Download all dependencies. Dependencies will be cached if go.mod and go.sum are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /collector ./main.go

# Start a new stage from scratch
FROM alpine:latest

# Install ca-certificates to ensure HTTPS connections work
RUN apk --no-cache add ca-certificates

# Set the current working directory inside the container
WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /collector .

# Copy the configs directory
COPY --from=builder /app/configs ./configs

# Copy the migrations directory
COPY --from=builder /app/migrations ./migrations

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
CMD ["./collector"]
