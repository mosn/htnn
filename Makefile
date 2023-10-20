SHELL = /bin/bash
OS = $(shell uname)

TARGET_SO       = libgolang.so
PROJECT_NAME    = mosn.io/moe
# Both images use glibc 2.31. Ensure libc in the images match each other.
BUILD_IMAGE     ?= golang:1.20-bullseye
PROXY_IMAGE     ?= envoyproxy/envoy:contrib-debug-dev

MAJOR_VERSION   = $(shell cat VERSION)
GIT_VERSION     = $(shell git log -1 --pretty=format:%h)

# Define a recursive wildcard function
rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$d/,$(2))$(filter $(subst *,%,$(2)),$d))

PROTOC = protoc
PROTO_FILES = $(call rwildcard,./plugins/,*.proto)
GO_TARGETS = $(patsubst %.proto,%.pb.go,$(PROTO_FILES))

TEST_OPTION ?= -gcflags="all=-N -l" -v


.PHONY: gen-proto
gen-proto: $(GO_TARGETS)
%.pb.go: %.proto
	protoc --proto_path=. --go_opt="paths=source_relative" --go_out=. --validate_out="lang=go,paths=source_relative:." -I ../protoc-gen-validate $<

.PHONY: test
test:
	go test ${TEST_OPTION} ./pkg/...
	go test ${TEST_OPTION} ./plugins/...

.PHONY: build-so-local
build-so-local:
	CGO_ENABLED=1 go build -tags so \
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X main.Version=${MAJOR_VERSION}(${GIT_VERSION})" \
		--buildmode=c-shared \
		-v -o ${TARGET_SO} \
		${PROJECT_NAME}/cmd/libgolang

.PHONY: build-so
build-so:
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		make build-so-local

.PHONY: run-demo
run-demo: build-so
	docker run --rm -v $(PWD)/etc/demo.yaml:/etc/demo.yaml \
		-v $(PWD)/libgolang.so:/etc/libgolang.so \
		-p 10000:10000 \
		${PROXY_IMAGE} \
		envoy -c /etc/demo.yaml --log-level debug
