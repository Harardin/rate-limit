ARG TARGET_DIR=/app
ARG GOBIN=/.bin

FROM golang:1.18.0-alpine

ARG TARGET_DIR
ARG GOBIN

RUN apk add --update make git musl-dev gcc libc-dev binutils-gold

ENV GO111MODULE=on
ENV GOPRIVATE=gitlab.com
ENV GOPATH=/go
WORKDIR /go/src/worker

COPY go.mod go.mod
COPY go.sum go.sum
COPY .netrc /root/.netrc
COPY . .
RUN go mod download

# Enable this if some tools needed in docker
# RUN GOBIN=$GOBIN make install-tools

RUN TARGET_DIR=$TARGET_DIR make build

ENTRYPOINT make watch

