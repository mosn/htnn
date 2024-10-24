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
ARG PROXY_BASE_IMAGE
# hadolint ignore=DL3006
FROM ${GO_BUILD_BASE_IMAGE} as builder
ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY
ENV GOPROXY=$GOPROXY

WORKDIR /htnn

# Copy the source

COPY go.* ./
RUN sed -i '/.\/site/d' go.work && sed -i '/.\/e2e/d' go.work && sed -i '/.\/tools/d' go.work
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
RUN make prebuild

COPY patch/switch-envoy-go-version.sh patch/switch-envoy-go-version.sh
COPY common.mk common.mk
# hadolint ignore=DL3003
RUN ./patch/switch-envoy-go-version.sh 1.29.5 && \
    cd plugins/ && \
    ENVOY_API_VERSION=1.29 make build-so-local

WORKDIR /istio
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -C pilot/cmd/pilot-agent -a -o /htnn/pilot-agent

# hadolint ignore=DL3006
FROM ${PROXY_BASE_IMAGE}

# Labels below are consumed by GitHub Container Registry
# See https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#labelling-container-images
LABEL org.opencontainers.image.source=https://github.com/mosn/htnn
LABEL org.opencontainers.image.description="This is image used in the HTNN data plane."
LABEL org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /htnn/pilot-agent /usr/local/bin/
COPY --from=builder /htnn/plugins/libgolang.so /etc/libgolang.so
CMD ["envoy"]
