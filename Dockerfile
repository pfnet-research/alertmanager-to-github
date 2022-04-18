# syntax = docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.18 AS base
WORKDIR /workspace
ENV CGO_ENABLED=0
COPY go.* .
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM golangci/golangci-lint:v1.45 AS lint-base
FROM base AS lint
RUN --mount=target=. \
    --mount=from=lint-base,src=/usr/bin/golangci-lint,target=/usr/bin/golangci-lint \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.cache/golangci-lint \
    golangci-lint run --timeout 10m0s ./...

FROM base AS unit-test
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -v ./...

FROM base AS build
ARG TARGETOS TARGETARCH
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/alertmanager-to-github .

FROM scratch AS export
COPY --from=build /out/alertmanager-to-github /

FROM --platform=$BUILDPLATFORM gcr.io/distroless/static:nonroot
COPY --from=build /out/alertmanager-to-github /
ENTRYPOINT ["/alertmanager-to-github"]
CMD ["start"]
