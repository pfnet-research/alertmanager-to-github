FROM golang:1.14 as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . /workspace

RUN go build -o bin/out .

FROM ubuntu:20.04
WORKDIR /
RUN apt-get update && apt-get install -y \
    ca-certificates \
 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /workspace/bin/out /usr/local/bin/alertmanager-to-github
ENTRYPOINT ["/usr/local/bin/alertmanager-to-github", "start"]

