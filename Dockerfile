# Stage 1: Build the Go binary
FROM golang:1.23.1-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

# Set the working directory
WORKDIR /GoAdBlock

# Copy Go modules files and download dependencies
COPY go.mod go.sum ./

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o GoAdBlock ./cmd/server/main.go

# Stage 2: Create a minimal runtime image
FROM alpine:latest

RUN adduser -D goadblock && mkdir /GoAdBlock
# Set the working directory
WORKDIR /GoAdBlock/

# Copy the binary from the builder stage
COPY --from=builder /GoAdBlock/GoAdBlock .

# Set execution permissions
RUN chmod +x GoAdBlock
EXPOSE 8080 53
# Command to run the app
CMD ["./GoAdBlock"]
