VERSION=$(shell git describe --tags --abbrev=0)

ifeq ($(VERSION),)
	VERSION="0.0.0"
endif

lint:
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.58.1 run

test:
	go test -cover -race -v ./...

.PHONY: build
build:
	mkdir -p ./build
	go build \
		-o ./build/pray ./cmd

image:
	docker build \
    		--tag brucexc/pray:$(VERSION) \
    		.

run:
	ENVIRONMENT=development go run ./cmd
