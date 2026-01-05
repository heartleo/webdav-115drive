FROM golang:1.24-alpine AS builder

WORKDIR /

ENV GOPROXY=https://goproxy.cn

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build github.com/heartleo/webdav-115drive

FROM alpine:3.22

WORKDIR /

COPY --from=builder /webdav-115drive .

CMD ["/webdav-115drive"]
