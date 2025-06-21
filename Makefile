GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

all: lint build

lint:
	golangci-lint run ./...

build:
	CGO_ENABLED=0 go build -o ./bin/regnotify-$(GOOS)-$(GOARCH) ./cmd/regnotify

docker:
	docker build -t localhost/evanebb/regnotify:latest .
