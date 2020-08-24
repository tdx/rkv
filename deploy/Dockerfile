FROM golang:1.15-alpine AS build
WORKDIR /go/src/rkv
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/rkvd ./cmd/rkvd
