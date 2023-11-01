SHELL = /bin/bash
OS = $(shell uname)

TARGET_SO       = libgolang.so
PROJECT_NAME    = mosn.io/moe
# Both images use glibc 2.31. Ensure libc in the images match each other.
BUILD_IMAGE     ?= golang:1.20-bullseye
# This is the envoyproxy/envoy:contrib-debug-dev fetched in 2023-10-27
# Use docker inspect --format='{{index .RepoDigests 0}}' envoyproxy/envoy:contrib-debug-dev
# to get the sha256 ID
PROXY_IMAGE     ?= envoyproxy/envoy@sha256:216c1c849c326ffad0a249de6886fd90c1364bbac193d5b1e36846098615071b
# The TEST_IMAGE doesn't need to be the same with BUILD_IMAGE
TEST_IMAGE      ?= golang:1.20-bullseye
# local build image doesn't have RepoDigests, use timestamp as tag to distinguish different images
DEV_TOOLS_IMAGE ?= moe-dev-tools:2023-10-23

MAJOR_VERSION   = $(shell cat VERSION)
GIT_VERSION     = $(shell git log -1 --pretty=format:%h)

# Define a recursive wildcard function
rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$d/,$(2))$(filter $(subst *,%,$(2)),$d))

PROTOC = protoc
PROTO_FILES = $(call rwildcard,./plugins/,*.proto)
GO_TARGETS = $(patsubst %.proto,%.pb.go,$(PROTO_FILES))

TEST_OPTION ?= -gcflags="all=-N -l" -v -race


.PHONY: gen-proto
gen-proto: build-dev-tools $(GO_TARGETS)
# format the generated Go code so the `fmt-go` task can pass
%.pb.go: %.proto
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} --user $(shell id -u) -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		protoc --proto_path=. --go_opt="paths=source_relative" --go_out=. --validate_out="lang=go,paths=source_relative:." \
			-I ../../protoc-gen-validate $<
	go run github.com/rinchsan/gosimports/cmd/gosimports@latest -w -local ${PROJECT_NAME} $@

# So far, the gomonkey library used in the test can't always run in Mac.
# We have a discussion about whether to run test in Docker or use uber-go/mock.
# The conclusion is, we prefer easier to write test to easier to run test.
.PHONY: unit-test
unit-test:
	docker run --rm -v $(shell go env GOPATH):/go -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${TEST_IMAGE} make unit-test-local

.PHONY: unit-test-local
unit-test-local:
	go test ${TEST_OPTION} $(shell go list ./... | grep -v tests/)

.PHONY: integration-test
integration-test:
	go test ${TEST_OPTION} ./tests/integration/...

.PHONY: build-so-local
build-so-local:
	CGO_ENABLED=1 go build -tags so \
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X main.Version=${MAJOR_VERSION}(${GIT_VERSION})" \
		--buildmode=c-shared \
		-v -o ${TARGET_SO} \
		${PROJECT_NAME}/cmd/libgolang

.PHONY: build-so
build-so:
	docker run --rm -v $(shell go env GOPATH):/go -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		make build-so-local

.PHONY: run-demo
run-demo:
	docker run --rm -v $(PWD)/etc/demo.yaml:/etc/demo.yaml \
		-v $(PWD)/libgolang.so:/etc/libgolang.so \
		-p 10000:10000 \
		${PROXY_IMAGE} \
		envoy -c /etc/demo.yaml --log-level debug

.PHONY: build-dev-tools
build-dev-tools:
	@if ! docker images ${DEV_TOOLS_IMAGE} | grep dev-tools > /dev/null; then \
		docker build --network=host --build-arg GOPROXY=${GOPROXY} -t ${DEV_TOOLS_IMAGE} -f tools/Dockerfile.dev ./tools; \
	fi

.PHONY: lint-go
lint-go:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --config=.golangci.yml

.PHONY: fmt-go
fmt-go:
	go mod tidy
	go run github.com/rinchsan/gosimports/cmd/gosimports@latest -w -local ${PROJECT_NAME} .

.PHONY: lint-spell
lint-spell: build-dev-tools
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		make lint-spell-local

.PHONY: lint-spell-local
lint-spell-local:
	codespell --skip '.git,.idea,test-envoy,go.mod,go.sum,*.svg' --check-filenames --check-hidden --ignore-words ./.ignore_words
