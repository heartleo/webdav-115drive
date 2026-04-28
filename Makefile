.PHONY: build fmt tidy vet lint

build:
	go build github.com/heartleo/webdav-115drive/cmd/webdav-115drive

fmt:
	go fmt ./...

tidy:
	go mod tidy

vet:
	go vet ./...

lint:
	golangci-lint run ./...
