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

include ../common.mk

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

PROXY_IMAGE ?= htnn/proxy:latest
PROXY_BASE_IMAGE ?= istio/proxyv2:$(ISTIO_VERSION)
CONTROLLER_IMAGE ?= htnn/controller:latest
CONTROLLER_BASE_IMAGE ?= docker.io/istio/pilot:$(ISTIO_VERSION)

.PHONY: build-so
build-so:
	cd ../plugins/ && make build-so

.PHONY: build-proxy-image
build-proxy-image: build-so
	cd .. && $(CONTAINER_TOOL) build -t ${PROXY_IMAGE} --build-arg GOPROXY=${GOPROXY} --build-arg PROXY_BASE_IMAGE=${PROXY_BASE_IMAGE} \
		-f manifest/image/dp/Dockerfile .

# If you wish to build the controller image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: build-controller-image
build-controller-image:
	cd .. && $(CONTAINER_TOOL) build -t ${CONTROLLER_IMAGE} \
		--build-arg GOPROXY=${GOPROXY} --build-arg CONTROLLER_BASE_IMAGE=${CONTROLLER_BASE_IMAGE} \
		-f manifest/image/cp/Dockerfile .

# PLATFORMS defines the target platforms for the image be built to provide support to multiple
# architectures. To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via CONTROLLER_IMAGE=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64
# TODO: support building proxy image with buildx
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' image/cp/Dockerfile > /tmp/Dockerfile.cross
	cd .. && $(CONTAINER_TOOL) buildx create --name project-v3-builder; \
	$(CONTAINER_TOOL) buildx use project-v3-builder && \
	$(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${CONTROLLER_IMAGE} \
		--build-arg GOPROXY=${GOPROXY} --build-arg CONTROLLER_BASE_IMAGE=${CONTROLLER_BASE_IMAGE} \
		-f /tmp/Dockerfile.cross . ; \
	$(CONTAINER_TOOL) buildx rm project-v3-builder
	rm /tmp/Dockerfile.cross

.PHONY: push-image
push-image:
	$(CONTAINER_TOOL) push ${CONTROLLER_IMAGE}
	$(CONTAINER_TOOL) push ${PROXY_IMAGE}
