# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
#
# SPDX-License-Identifier: Apache-2.0

# Build the manager binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT='not set'
ARG GIT_TAG=development
ENV GIT_COMMIT=$GIT_COMMIT
ENV GIT_TAG=$GIT_TAG
ENV CPU_ARCH=$TARGETARCH

RUN echo $TARGETARCH

WORKDIR /build

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# build app
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GO111MODULE=on \
    go build -ldflags="-X 'main.version=$GIT_TAG' -X 'main.commit=$GIT_COMMIT'" -o /out/prysm ./cmd/main.go


FROM alpine
LABEL source_repository="https://github.com/cobaltcore-dev/prysm"
# Install smartctl
RUN apk add --no-cache smartmontools nvme-cli

# copy app bianry
COPY --from=builder /out/prysm /bin/prysm
# RUN chown 1001:1001 /bin/prysm

WORKDIR /bin
# USER 1001
ENTRYPOINT ["/bin/prysm"]
