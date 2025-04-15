# Auth Service for AI Shopper

This service provides authentication and authorization functionality for the AI Shopper application using OAuth2.

## Overview

The auth service implements OAuth2 authentication using the go-oauth2 library. It supports:

- Authorization code flow
- JWT-based access tokens
- Redis-backed token storage

## Architecture

The service consists of:

- **OAuth2 Server**: Handles authentication and authorization flows
- **Token Storage**: Redis-based storage for tokens with Lua scripts for atomic operations
- **User Management**: User authentication and management
- **Client Management**: OAuth client registration and management
- **Login page**: Powered by tailwind css

## Configuration

The service can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| REDIS_HOST | Redis host | localhost |
| REDIS_PORT | Redis port | 6379 |
| REDIS_USER | Redis username | default |
| REDIS_PASSWORD | Redis password | password |
| JWT_SECRET | Secret for JWT signing | your-secret-key |
| CODE_TTL | Authorization code TTL in seconds | 300 |
| ACCESS_TTL | Access token TTL in seconds | 3600 |
| REFRESH_TTL | Refresh token TTL in seconds | 86400 |

## Development

### Prerequisites

- Go 1.16+
- Redis 7.0+
- MySQL 8.0+

### Running Locally

1. Start Redis and MySQL using Docker Compose:
   ```
   docker-compose -f docker-compose-infra.yml up -d
   ```

2. Build the service with 
    ```
    docker-compose build auth
    ```

3. Run the service with 
    ```
    docker-compose up
    ```

### Testing

Run tests with:
```
go test ./...
```

Requires the testing infra to be up and running. Run testing infra with
```
docker-compose -f docker-compose-test.yml up -d
```

### Token Storage

Tokens are stored in Redis with the following structure:
- `code:{userID}:{code}` - Authorization codes
- `access:{userID}:{access}` - Access tokens
- `refresh:{userID}:{refresh}` - Refresh tokens

Lua scripts are used to ensure atomic operations when creating and managing tokens. Lua SHA is stored in REDIS with the key `SHA:createScript`.