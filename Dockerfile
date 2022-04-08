# syntax = docker/dockerfile:1
FROM golang:1.18 AS base
WORKDIR /workspace
ENV CGO_ENABLED=0
COPY go.* .
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM base AS unit-test
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -v ./...

FROM base AS build
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/alertmanager-to-github .

FROM scratch AS export
COPY --from=build /out/alertmanager-to-github /

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/alertmanager-to-github /
ENTRYPOINT ["/alertmanager-to-github"]
CMD ["start"]
