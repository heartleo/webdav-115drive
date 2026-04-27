FROM golang:1.25-alpine AS builder

WORKDIR /

ENV GOPROXY=https://goproxy.cn

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o webdav-115drive github.com/heartleo/webdav-115drive/cmd/webdav-115drive

FROM alpine:3.22

WORKDIR /

COPY --from=builder /webdav-115drive .

CMD ["/webdav-115drive"]
