# Copyright The HTNN Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Dockerfile has specific requirement: https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG CONTROLLER_BASE_IMAGE
# Build the manager binary
FROM golang:1.21 as builder
ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY
ENV GOPROXY=$GOPROXY

WORKDIR /workspace

# Copy the source

COPY go.* ./
RUN sed -i '/.\/site/d' go.work # remove this in release
# use single layer to copy only works on files
COPY api/ api/
COPY types/ types/
COPY pkg/ pkg/
COPY plugins/ plugins/
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download
COPY controller/ controller/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN cd controller && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o /workspace/manager cmd/main.go

FROM ${CONTROLLER_BASE_IMAGE}
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
