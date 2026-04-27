.PHONY: build fmt tidy vet

build:
	go build github.com/heartleo/webdav-115drive/cmd/webdav-115drive

fmt:
	go fmt ./...

tidy:
	go mod tidy

vet:
	go vet ./...
