RUN_ARGS?=

.PHONY: run tidy fmt

run:
	go run ./cmd/api $(RUN_ARGS)

tidy:
	go mod tidy

fmt:
	gofmt -s -w .

# Swagger documentation generation
.PHONY: swagger
swagger:
	$(shell go env GOPATH)/bin/swag init -g cmd/api/main.go -o docs

# Delve (Go debugger)
.PHONY: dlv-install dlv-debug dlv-headless

dlv-install:
	go install github.com/go-delve/delve/cmd/dlv@latest

# Start debug session locally (interactive UI)
dlv-debug:
	dlv debug ./cmd/api

# Headless mode for IDE attach (listens on :2345)
dlv-headless:
	dlv debug ./cmd/api --headless --listen=:2345 --api-version=2 --accept-multiclient
