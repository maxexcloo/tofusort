# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24 AS builder

ARG TARGETARCH
ARG TARGETOS

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Cross-compile on the runner's native platform instead of under emulation.
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH GOOS=$TARGETOS go build -o /tofusort -trimpath ./cmd/tofusort

FROM scratch

COPY --from=builder /tofusort /tofusort

ENTRYPOINT ["/tofusort"]
