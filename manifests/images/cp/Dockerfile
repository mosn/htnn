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
ARG GO_BUILD_BASE_IMAGE
ARG CONTROLLER_BASE_IMAGE
# hadolint ignore=DL3006
FROM ${GO_BUILD_BASE_IMAGE} as builder
ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY
ENV GOPROXY=$GOPROXY

WORKDIR /workspace

# Copy the source

COPY go.* ./
RUN sed -i 's/.\/site/.\/external\/istio/' go.work && sed -i '/.\/e2e/d' go.work && sed -i '/.\/tools/d' go.work
# use single layer to copy only works on files, so we have to list all the directories
COPY api/ api/
COPY types/ types/
COPY plugins/ plugins/
COPY controller/ controller/
# Copy files used by `make prebuild`
COPY Makefile common.mk ./
COPY patch/ patch/
# Download istio to external/istio
# Note that if we switch to non-debian base image these commands will break...
RUN cat <<_EOF > /etc/apt/sources.list.d/debian.sources
Types: deb
URIs: https://mirrors.tuna.tsinghua.edu.cn/debian
Suites: bookworm bookworm-updates bookworm-backports
Components: main contrib non-free non-free-firmware
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg
_EOF

# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y patch --no-install-recommends
RUN TARGET_ISTIO_DIR=$(pwd)/external/istio make prebuild
WORKDIR /workspace/external/istio
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -C pilot/cmd/pilot-discovery -a -o /workspace/pilot-discovery

# hadolint ignore=DL3006
FROM ${CONTROLLER_BASE_IMAGE}

# Labels below are consumed by GitHub Container Registry
# See https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#labelling-container-images
LABEL org.opencontainers.image.source=https://github.com/mosn/htnn
LABEL org.opencontainers.image.description="This is image used in the HTNN control plane."
LABEL org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /workspace/pilot-discovery /usr/local/bin/
