# HTTP Echo Server

A simple web application that captures and displays HTTP requests
in real-time. The application consists of a Go backend server that
stores requests in SQLite and a React frontend that displays the
requests in a clean, modern UI.

## Prerequisites

- Go 1.24 or later
- Node.js 22 or later
- Yarn 4.7.0 or later (using Plug'n'Play)
- SQLite 3
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

3. Format the code:
   ```bash
   task format              # Format both frontend and backend
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
- Stores the request with timestamp and headers in SQLite
- Returns a success message

Example:
```bash
curl -X POST http://localhost:8025/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, World!"}'
```

### GET /echo
- Returns all stored requests in reverse chronological order from SQLite
- Format: JSON array of request objects
- Returns an empty array if no requests are stored

Example:
```bash
curl http://localhost:8025/echo
```

## License

[MIT License](LICENSE)
