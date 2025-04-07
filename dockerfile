# stage 1: build the tailwind css for the login page
FROM node:18-alpine AS tailwind-builder

WORKDIR /app

COPY web ./web
RUN npm install --prefix ./web
RUN echo '@tailwind base; @tailwind components; @tailwind utilities;' > ./web/input.css

# Create static/css directory
RUN mkdir -p ./web/static/css

# Run tailwind with verbose output
RUN cd web && npx tailwindcss -i ./input.css -o ./static/css/output.css --verbose

# Debug: List the contents of the web directory
RUN ls -la ./web
RUN ls -la ./web/static/css

# stage 2: build the Go source code
FROM golang:1.24.0 AS go-builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

#Copy the tailwind css
COPY --from=tailwind-builder /app/web/static/css/output.css ./web/static/css/output.css

# Build the application
RUN GOOS=linux go build -o main .

#Stage 3: build the docker image
FROM golang:1.24.0

WORKDIR /app

# Copy the built go binary
COPY --from=go-builder /app/main .
#COPY the static files for tailwind css
COPY --from=go-builder /app/web/static ./web/static
COPY --from=go-builder /app/web/templates ./web/templates

# Expose port 9096
EXPOSE 9096

# Run the application
CMD ["./main"]

