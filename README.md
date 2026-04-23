# Insight Tool

A modern web application built with Go and React that analyzes web pages to extract metadata, heading structures, and link statistics 

■ HTML version (e.g., HTML5, HTML 4.01)

■ Page title

■ Count of headings per level (h1 through h6)

■ Number of internal and external links

■ Count of inaccessible links

■ Whether the page contains a login form

## 🚀 Getting Started

### Prerequisites
- **Go 1.25+** (The project uses modern Go features and toolchains)

### Run the Application
1. Clone the repository and navigate to the project root.
2. Start the server:
   ```bash
   go run cmd/server/main.go
   ```
3. The application will be available at `http://localhost:8080`.

### Using Makefile
For convenience, a `Makefile` is provided for common tasks:
- `make run`: Run the application locally.
- `make build`: Compile the binary.
- `make test`: Run all unit tests.
- `make docker-build`: Build the Docker image.
- `make docker-run`: Run the application inside a Docker container.

### Using Docker
1. Build the image:
   ```bash
   docker build -t insight-tool .
   ```
2. Run the container:
   ```bash
   docker run -p 8080:8080 insight-tool
   ```

### Run Tests
To verify the analysis logic and HTTP handlers:
```bash
go test ./internal/...
```

## 🏗️ Design Decisions & Assumptions

### Technical Choices
- **Tokenizer over DOM**: I chose `html.Tokenizer` from `golang.org/x/net/html` for parsing. Unlike a DOM-based approach that loads the entire page into memory, the tokenizer processes the document as a stream. This ensures **O(1) memory complexity**, making the tool highly efficient even for extremely large web pages.
- **Concurrent Link Checking**: The tool checks the accessibility of all unique links in parallel using Go's concurrency primitives (`goroutines` and `channels`). This significantly reduces the total analysis time.
- **Embedded Frontend**: The React UI is embedded into the Go binary using `embed.FS` and `io/fs`. This results in a single, portable executable that doesn't require a separate web server or build step for the frontend.
- **React via CDN**: To keep the project simple and avoid a complex Node.js build pipeline, React and Tailwind-like styles are loaded via CDN, allowing for a fast and modern UI without `npm` dependencies.

### Deployment & Containerization
I have included a `Dockerfile` and a `Makefile` to demonstrate how the application can be easily containerized and deployed in a production environment. The Dockerfile uses a **multi-stage build** to ensure the final image is as lightweight as possible while including necessary root certificates for secure HTTPS analysis.

### Assumptions
- **Modern HTML**: The tool assumes most modern pages use HTML5 but provides specific detection for legacy versions (HTML 4.01, XHTML) based on DOCTYPE declarations.
- **Accessibility**: A link is considered "inaccessible" if it returns a non-2xx/3xx status code (e.g., 404, 403, 500) or fails to connect within a 10-second timeout.
- **Login Form Detection**: The tool identifies a login form by looking for a `<form>` containing at least one `password` input and a `submit` mechanism.

## 🛠️ Project Structure
- `cmd/server/`: Main entry point and server configuration.
- `internal/analyzer/`: Core business logic for HTML parsing and link analysis.
- `internal/handler/`: HTTP API layer and request/response handling.
- `cmd/server/static/`: Frontend assets (HTML/React).

## 🔮 Future Improvements
- **Caching**: Implement a TTL-based cache (e.g., Redis or in-memory) to avoid re-analyzing the same URL multiple times within a short window.
- **Rate Limiting**: Add per-IP rate limiting to prevent abuse of the analysis endpoint.
- **SSRF Protection**: Add a blocklist for internal IP ranges (127.0.0.1, 10.0.0.0/8, etc.) to prevent the server from being used to probe internal networks.
- **Exporting**: Add the ability to export analysis results as PDF or CSV.
- **Monitoring & Observability**: Integrate Prometheus metrics and structured logging to track analysis latency, error rates, and system health.
