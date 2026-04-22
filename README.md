# Home24-Tech Analyzer

A simple Go application with a React-based UI to analyze web content.

## Structure
- `cmd/server/`: Entry point, embeds static files and starts the HTTP server.
- `internal/analyzer/`: Core analysis logic.
- `internal/handler/`: HTTP handler for the API.
- `static/`: Frontend assets (React via CDN).

## Getting Started
### Prerequisite
- [Go](https://golang.org/dl/) (1.18+)

### Run the application
```bash
go run cmd/server/main.go
```
The application will be available at `http://localhost:8080`.

### Run tests
```bash
go test ./internal/...
```

Forbidden Access 403 fixed --> https://www.home24.de