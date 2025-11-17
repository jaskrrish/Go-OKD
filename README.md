# Go-OKD Backend

A basic Go backend API server with RESTful endpoints.

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go          # Application entry point
├── internal/
│   ├── handlers/
│   │   └── handlers.go      # HTTP request handlers
│   └── models/
│       └── user.go          # Data models
├── pkg/
│   └── utils/               # Utility functions
├── go.mod                   # Go module dependencies
└── README.md
```

## Features

- RESTful API endpoints
- Health check endpoint
- User management (mock data)
- Request logging middleware
- Configurable server timeouts

## Prerequisites

- Go 1.19 or higher

## Installation

1. Clone the repository
2. Navigate to the project directory
3. Install dependencies (if any):
   ```bash
   go mod download
   ```

## Running the Server

Start the server with default settings (port 8080):

```bash
go run cmd/api/main.go
```

Or specify a custom port:

```bash
PORT=3000 go run cmd/api/main.go
```

## API Endpoints

### Root
- **GET** `/`
  - Returns welcome message and API information

### Health Check
- **GET** `/health`
  - Returns server health status

### Users
- **GET** `/api/v1/users`
  - Returns list of users (mock data)

- **POST** `/api/v1/users`
  - Creates a new user
  - Request body:
    ```json
    {
      "username": "string",
      "email": "string"
    }
    ```

## Testing the API

Using curl:

```bash
# Get welcome message
curl http://localhost:8080/

# Health check
curl http://localhost:8080/health

# Get users
curl http://localhost:8080/api/v1/users

# Create user
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username":"test_user","email":"test@example.com"}'
```

## Building for Production

Build the binary:

```bash
go build -o bin/api cmd/api/main.go
```

Run the binary:

```bash
./bin/api
```

## Development

The server includes request logging middleware that logs all incoming requests and their completion time.

## Future Enhancements

- Database integration (PostgreSQL/MongoDB)
- Authentication & Authorization
- Input validation
- Error handling middleware
- Unit and integration tests
- Docker support
- CI/CD pipeline
