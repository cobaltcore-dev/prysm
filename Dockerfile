# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
#
# SPDX-License-Identifier: Apache-2.0

# ============================================================================
# Multi-stage Dockerfile for Prysm
#
# Default build (CGO_ENABLED=0, alpine, no ceph):
#   docker build -t prysm .
#
# Build with Ceph/RADOS support (pg-probe producer):
#   docker build --target runtime-ceph -t prysm-ceph .
#
# ============================================================================

# --- Stage: Base builder (shared Go module cache) ---
FROM golang:1.26-alpine AS builder-base
ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT='not set'
ARG GIT_TAG=development

ENV GIT_COMMIT=$GIT_COMMIT
ENV GIT_TAG=$GIT_TAG

WORKDIR /build

# Copy Go module manifests and cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source tree
COPY . .

# --- Stage: Build WITHOUT Ceph (default, static binary) ---
FROM builder-base AS builder-noceph

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -ldflags="-X 'main.version=$GIT_TAG' -X 'main.commit=$GIT_COMMIT'" \
    -o /out/prysm ./cmd/main.go

# --- Stage: Build WITH Ceph (CGO + librados, includes pg-probe) ---
FROM builder-base AS builder-ceph

# Install Ceph development libraries and build dependencies
RUN apk add --no-cache \
        gcc \
        musl-dev \
        linux-headers \
        ceph19-dev \
        librados19

RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -tags ceph \
    -ldflags="-X 'main.version=$GIT_TAG' -X 'main.commit=$GIT_COMMIT'" \
    -o /out/prysm ./cmd/main.go

# --- Stage: Runtime (alpine, no Ceph) ---
FROM alpine:3.21 AS runtime-noceph
LABEL source_repository="https://github.com/cobaltcore-dev/prysm"

RUN apk add --no-cache ca-certificates smartmontools nvme-cli

COPY --from=builder-noceph /out/prysm /bin/prysm

WORKDIR /bin
ENTRYPOINT ["/bin/prysm"]

# --- Stage: Runtime (alpine, with Ceph libs for pg-probe) ---
FROM alpine:3.21 AS runtime-ceph
LABEL source_repository="https://github.com/cobaltcore-dev/prysm"

RUN apk add --no-cache \
        ca-certificates \
        librados19 \
        smartmontools \
        nvme-cli

COPY --from=builder-ceph /out/prysm /bin/prysm

WORKDIR /bin
ENTRYPOINT ["/bin/prysm"]

# --- Default target (standard alpine image without Ceph) ---
FROM runtime-noceph
