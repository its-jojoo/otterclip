APP_NAME=otterclip

.PHONY: fmt vet test lint check run

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./... -race

lint: fmt vet

check: lint test

run:
	go run ./cmd/$(APP_NAME)
