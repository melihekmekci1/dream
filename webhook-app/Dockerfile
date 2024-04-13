# Use a specific version of Go that meets your requirements
FROM golang:1.21-alpine as builder

# Set up necessary packages
RUN apk add --no-cache git gcc musl-dev

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webhook-server .

# Use a distroless base image for the final stage
FROM gcr.io/distroless/static-debian11

WORKDIR /

# Copy the binary from the builder stage
COPY --from=builder /app/webhook-server .

# Command to run
ENTRYPOINT ["/webhook-server"]
