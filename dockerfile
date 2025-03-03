# Use official Golang image as base
FROM golang:1.24.0

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main .

# Expose port 9096
EXPOSE 9096

# Run the application
CMD ["./main"]

