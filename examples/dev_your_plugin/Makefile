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

SHELL = /bin/bash

TARGET_SO       = libgolang.so
PROJECT_NAME    = mosn.io/htnn/dev_your_plugin
# Both images use glibc 2.31. Ensure libc in the images match each other.
BUILD_IMAGE     ?= golang:1.21-bullseye
# This is the envoyproxy/envoy:contrib-v1.29.2
# Use docker inspect --format='{{index .RepoDigests 0}}' envoyproxy/envoy:contrib-v1.29.2
# to get the sha256 ID
PROXY_IMAGE     ?= envoyproxy/envoy@sha256:c47136604751274b30fa7a89132314b8e3586d54d8f8cc30d7a911a9ecc5e11c
DEV_TOOLS_IMAGE ?= ghcr.io/mosn/htnn-dev-tools:2024-03-05

PROTOC = protoc

# Define a recursive wildcard function
rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$d/,$(2))$(filter $(subst *,%,$(2)),$d))
PROTO_FILES = $(call rwildcard,./,*.proto)
GO_TARGETS = $(patsubst %.proto,%.pb.go,$(PROTO_FILES))

MOUNT_GOMOD_CACHE = -v $(shell go env GOPATH):/go
ifeq ($(IN_CI), true)
	# Mount go mod cache in the CI environment will cause 'Permission denied' error
	# when accessing files on host in later phase because the mounted directory will
	# have files which is created by the root user in Docker.
	# Run as low privilege user in the Docker doesn't
	# work because we also need root to create /.cache in the Docker.
	MOUNT_GOMOD_CACHE =
endif

.PHONY: dev-tools
dev-tools:
	@if ! docker images ${DEV_TOOLS_IMAGE} | grep dev-tools > /dev/null; then \
		docker pull ${DEV_TOOLS_IMAGE}; \
	fi

.PHONY: gen-proto
gen-proto: dev-tools $(GO_TARGETS)
%.pb.go: %.proto
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} --user $(shell id -u) -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		protoc --proto_path=. --go_opt="paths=source_relative" --go_out=. --validate_out="lang=go,paths=source_relative:." \
			-I /go/src/protoc-gen-validate $<

.PHONY: build-so-local
build-so-local:
	CGO_ENABLED=1 go build -tags so \
		--buildmode=c-shared \
		-v -o ${TARGET_SO} \
		${PROJECT_NAME}/cmd/libgolang

# As the tasks below mount the GOPATH into the docker container, please make sure you don't have Go binary put into the GOPATH
# which will override the one provides by the docker image.

.PHONY: build-so
build-so:
	docker run --rm $(MOUNT_GOMOD_CACHE) -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		make build-so-local

.PHONY: unit-test-local
unit-test-local:
	go test -v ./plugins/...

.PHONY: unit-test
unit-test:
	docker run --rm $(MOUNT_GOMOD_CACHE) -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		make unit-test-local

.PHONY: build-test-so-local
build-test-so-local:
	CGO_ENABLED=1 go build -tags so \
		--buildmode=c-shared \
		-cover -covermode=atomic -coverpkg=${PROJECT_NAME}/... \
		-v -o ./tests/integration/${TARGET_SO} \
		${PROJECT_NAME}/cmd/libgolang

.PHONY: build-test-so
build-test-so:
	docker run --rm $(MOUNT_GOMOD_CACHE) -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		make build-test-so-local

# This task can only be run in the CI, because we have `go.work` in the root directory and this module
# is not in it. We don't add this module to the `go.work` to ensure the `go.mod` and `go.sum` files in
# this module are always up-to-date.
# In the CI environment, we remove the `go.work`. If you want to run the integration test locally,
# you can remove the `go.work` and run `make integration-test`, or copy this module to a directory
# which doesn't have `go.work` in its ancestors.
.PHONY: integration-test
integration-test: build-test-so
	test -d /tmp/htnn_coverage && rm -rf /tmp/htnn_coverage || true
	go test -v ./tests/integration/...

.PHONY: run-plugin
run-plugin:
	docker run --name dev_your_plugin --rm -d -v $(PWD)/etc/demo.yaml:/etc/demo.yaml \
		-v $(PWD)/libgolang.so:/etc/libgolang.so \
		-p 10000:10000 \
		${PROXY_IMAGE} \
		envoy -c /etc/demo.yaml

.PHONY: stop-plugin
stop-plugin:
	docker stop dev_your_plugin
