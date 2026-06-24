.PHONY: build test run docker-build docker-run lint clean

APP_NAME := sre-agent

build:
	go build -o bin/$(APP_NAME) ./cmd/agent

test:
	go test ./... -v -count=1

run:
	go run ./cmd/agent

docker-build:
	docker build -t $(APP_NAME):dev .

docker-run:
	docker compose up -d

lint:
	golangci-lint run ./...

eval:
	go run ./tests/eval/runner.go

clean:
	rm -rf bin/
