.PHONY: build run test docker-build docker-run clean

BINARY_NAME=server

build:
	go build -o $(BINARY_NAME) ./cmd/server/main.go

run:
	go run ./cmd/server/main.go

test:
	go test ./internal/...

docker-build:
	docker build -t insight-tool .

docker-run:
	docker run -p 8080:8080 insight-tool

clean:
	rm -f $(BINARY_NAME)
	go clean
