# Start with a smaller Go base image
FROM golang:1.18-alpine AS builder

# Install git, gcc, musl-dev (if your project needs CGO, gcc, and musl-dev might be necessary)
RUN apk add --no-cache git gcc musl-dev

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook-server .

# Use a smaller image to run the server
FROM alpine:latest

# Add CA certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/webhook-server .

# Copy the TLS certificates
COPY webhook-server.crt webhook-server.key ./

# Expose the port your server listens on
EXPOSE 443

# Run the webhook server binary
CMD ["./webhook-server"]
