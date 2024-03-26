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
PROTO_FILES = $(call rwildcard,./,*.proto)
GO_TARGETS = $(patsubst %.proto,%.pb.go,$(PROTO_FILES))

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

.PHONY: build-so
build-so:
	docker run --rm -v $(shell go env GOPATH):/go -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		bash -c "git config --global --add safe.directory '*' && make build-so-local"

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
