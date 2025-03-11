# Stage 1: Build the Go binary
ARG GO_VERSION=1.23
ARG PLATFORM=linux/arm64

# Use build arguments in the builder stage
FROM --platform=$PLATFORM golang:$GO_VERSION-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

# Set the working directory
WORKDIR /GoAdBlock

# Copy Go modules files and download dependencies
COPY go.mod go.sum ./

# Copy the source code
COPY . .

# Build the Go application
RUN GOOS=linux GOARCH=arm64 go build -o goadblock ./cmd/server/main.go

# Stage 2: Create a minimal runtime image


FROM --platform=$PLATFORM alpine:latest


RUN adduser -D goadblock && mkdir /GoAdBlock
# Set the working directory
WORKDIR /GoAdBlock/

# Copy the binary from the builder stage
COPY --from=builder /GoAdBlock/goadblock .

# Set execution permissions
RUN chmod +x goadblock
EXPOSE 8080 53
# Command to run the app
CMD ["./goadblock"]
