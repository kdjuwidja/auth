# Auth Service for shopper app

This service provides authentication and authorization functionality for the AI Shopper application using OAuth2.

## Overview

The auth service implements OAuth2 authentication using the go-oauth2 library. It supports:

- Authorization code flow
- JWT-based access tokens
- Redis-backed token storage with configurable limit on the number of issued tokens

## Development

### Prerequisites

- Go 1.16+
- Redis 7.0+
- MySQL 8.0+

### Running Locally
You can obtain the docker compose files from [ai_shopper_docker_compose](https://github.com/kdjuwidja/ai_shopper_docker_compose)

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
- `code:{userID}:{code}` - Authorization codes (5 minutes TTL)
- `access:{userID}:{access}` - Access tokens (1 hour TTL)
- `refresh:{userID}:{refresh}` - Refresh tokens (24 hours TTL)

Lua scripe is used to ensure atomic operations when creating and managing tokens to enforce configurable limit on tokens per user. See `./lua/create.lua` for implementation details.