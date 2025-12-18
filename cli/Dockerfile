FROM golang:1.25-alpine AS builder
ARG VERSION=dev
ARG COMMIT_HASH=unknown

WORKDIR /workspace
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY api/ api/
COPY cli/ cli/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo \
    -ldflags "-X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH}" \
    -o koncli ./cli/

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /workspace/koncli .

ENTRYPOINT ["./koncli"]