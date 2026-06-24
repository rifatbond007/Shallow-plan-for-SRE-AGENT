.PHONY: build test run docker-build docker-run lint eval clean helm-lint helm-install helm-uninstall

APP_NAME := sre-agent
NAMESPACE := default

build:
	go build -o bin/$(APP_NAME) ./cmd/agent

test:
	go test ./... -v -count=1

run:
	go run ./cmd/agent

docker-build:
	docker build -t $(APP_NAME):dev .

docker-run:
	docker compose -f configs/docker-compose.yml up -d

docker-stop:
	docker compose -f configs/docker-compose.yml down

lint:
	golangci-lint run ./...

eval:
	go run ./tests/eval/runner.go

helm-lint:
	helm lint deploy/charts/$(APP_NAME)

helm-install:
	helm upgrade --install $(APP_NAME) deploy/charts/$(APP_NAME) --namespace $(NAMESPACE) --create-namespace

helm-uninstall:
	helm uninstall $(APP_NAME) --namespace $(NAMESPACE)

clean:
	rm -rf bin/
