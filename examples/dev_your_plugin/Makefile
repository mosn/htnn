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
PROJECT_NAME    = mosn.io/htnn
# Both images use glibc 2.31. Ensure libc in the images match each other.
BUILD_IMAGE     ?= golang:1.21-bullseye
PROXY_IMAGE     ?= envoyproxy/envoy:contrib-debug-dev


.PHONY: build-so-local
build-so-local:
	CGO_ENABLED=1 go build -tags so \
		--buildmode=c-shared \
		-v -o ${TARGET_SO} \
		${PROJECT_NAME}/dev_your_plugin/cmd/libgolang

.PHONY: build-so
build-so:
	docker run --rm -v $(shell go env GOPATH):/go -v $(PWD)/../..:/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME}/examples/dev_your_plugin \
		-e GOPROXY \
		${BUILD_IMAGE} \
		make build-so-local

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
