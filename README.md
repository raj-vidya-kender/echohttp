# HTTP Echo Server

A simple web application that captures and displays HTTP requests
in real-time. The application consists of a Go backend server that
stores requests in memory and a React frontend that displays the
requests in a clean, modern UI.

## Prerequisites

- Go 1.24 or later
- Node.js and Yarn
- Task (task runner) - [Installation Guide](https://taskfile.dev/installation/)

## Building the Project

The project uses Taskfile for build automation. Here are the available commands:

1. Clean the project:
   ```bash
   task clean
   ```

2. Build everything (both frontend and backend):
   ```bash
   task build
   ```

## Running the Application

1. Start the server:
   ```bash
   task run
   ```
   The server will start on port 8025 by default. You can change this by setting the `HTTP_PORT` environment variable:
   ```bash
   HTTP_PORT=3000 task run
   ```

2. Open your browser and visit:
   ```
   http://localhost:8025
   ```

## API Endpoints

### POST /echo
- Accepts any JSON payload
- Stores the request with timestamp and headers
- Returns a success message

Example:
```bash
curl -X POST http://localhost:8025/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, World!"}'
```

### GET /echo
- Returns all stored requests in reverse chronological order
- Format: JSON array of request objects

Example:
```bash
curl http://localhost:8025/echo
```

## License

[MIT License](LICENSE)
