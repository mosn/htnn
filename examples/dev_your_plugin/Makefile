SHELL = /bin/bash

TARGET_SO       = libgolang.so
PROJECT_NAME    = mosn.io/moe
# Both images use glibc 2.31. Ensure libc in the images match each other.
BUILD_IMAGE     ?= golang:1.20-bullseye
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
	docker run --rm -v $(PWD)/etc/demo.yaml:/etc/demo.yaml \
		-v $(PWD)/libgolang.so:/etc/libgolang.so \
		-p 10000:10000 \
		${PROXY_IMAGE} \
		envoy -c /etc/demo.yaml
