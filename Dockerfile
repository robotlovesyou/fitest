#syntax=docker/dockerfile:1.2
FROM golang:1.18-buster AS builder

WORKDIR /go/src/app
ADD . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN make install

FROM debian:10-slim

RUN apt-get update && apt-get upgrade -y && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/bin/users /users
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/users"]
