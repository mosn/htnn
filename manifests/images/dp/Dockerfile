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

# Dockerfile has specific requirement to put this ARG at the beginning:
# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG PROXY_BASE_IMAGE
FROM golang:1.21 as builder
ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY
ENV GOPROXY=$GOPROXY

WORKDIR /workspace

# Copy the source

COPY go.* ./
COPY api/ api/
COPY types/ types/
COPY plugins/ plugins/
COPY controller/ controller/
# Remember to run `make prebuild` before building the image
COPY external/istio/ external/istio/
WORKDIR /workspace/external/istio
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -C pilot/cmd/pilot-agent -a -o /workspace/pilot-agent

# hadolint ignore=DL3006
FROM ${PROXY_BASE_IMAGE}

# Labels below are consumed by GitHub Container Registry
# See https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#labelling-container-images
LABEL org.opencontainers.image.source=https://github.com/mosn/htnn
LABEL org.opencontainers.image.description="This is image used in the HTNN data plane."
LABEL org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /workspace/pilot-agent /usr/local/bin/
COPY plugins/libgolang.so /etc/libgolang.so
CMD ["envoy"]
