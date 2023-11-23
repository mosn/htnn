SHELL = /bin/bash
OS = $(shell uname)
IN_CI ?=

TARGET_SO       = libgolang.so
PROJECT_NAME    = mosn.io/moe
# Both images use glibc 2.31. Ensure libc in the images match each other.
BUILD_IMAGE     ?= golang:1.20-bullseye
# This is the envoyproxy/envoy:contrib-debug-dev fetched in 2023-11-22
# Use docker inspect --format='{{index .RepoDigests 0}}' envoyproxy/envoy:contrib-debug-dev
# to get the sha256 ID
PROXY_IMAGE     ?= envoyproxy/envoy@sha256:1fa13772ad01292fdbd73541717ef1a65fcdb2350bf13c173bddb10bf1f36c7c
# local build image doesn't have RepoDigests, use timestamp as tag to distinguish different images
DEV_TOOLS_IMAGE ?= moe-dev-tools:2023-10-23

MAJOR_VERSION   = $(shell cat VERSION)
GIT_VERSION     = $(shell git log -1 --pretty=format:%h)

# Define a recursive wildcard function
rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$d/,$(2))$(filter $(subst *,%,$(2)),$d))

PROTOC = protoc
PROTO_FILES = $(call rwildcard,./,*.proto)
GO_TARGETS = $(patsubst %.proto,%.pb.go,$(PROTO_FILES))

TEST_OPTION ?= -gcflags="all=-N -l" -v -race

MOUNT_GOMOD_CACHE = -v $(shell go env GOPATH):/go
ifeq ($(IN_CI), true)
	# Mount go mod cache in the CI environment will cause 'Permission denied' error
	# when accessing files on host in later phase because the mounted directory will
	# have files which is created by the root user in Docker.
	# Run as low privilege user in the Docker doesn't
	# work because we also need root to create /.cache in the Docker.
	MOUNT_GOMOD_CACHE =
endif

.PHONY: gen-proto
gen-proto: build-dev-tools $(GO_TARGETS)
# format the generated Go code so the `fmt-go` task can pass
%.pb.go: %.proto
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} --user $(shell id -u) -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		protoc --proto_path=. --go_opt="paths=source_relative" --go_out=. --validate_out="lang=go,paths=source_relative:." \
			-I ../../protoc-gen-validate $<
	go run github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8 -w -local ${PROJECT_NAME} $@

.PHONY: unit-test
unit-test:
	go test ${TEST_OPTION} $(shell go list ./... | grep -v tests/ | grep -v controller/ )

# We can't specify -race to `go build` because it seems that
# race detector assumes that the executable is loaded around the 0 address. When loaded by the Envoy,
# the race detector will allocate memory out of 48bits address which is not allowed in x64.
.PHONY: build-test-so-local
build-test-so-local:
	CGO_ENABLED=1 go build -tags so \
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X main.Version=${MAJOR_VERSION}(${GIT_VERSION})" \
		--buildmode=c-shared \
		-v -o tests/integration/plugins/${TARGET_SO} \
		${PROJECT_NAME}/tests/integration/plugins/libgolang

# Go 1.19+ adds vcs check which will cause error "fatal: detected dubious ownership in repository at '...'".
# So here we disable the error via git configuration when running inside Docker.
.PHONY: build-test-so
build-test-so:
	docker run --rm ${MOUNT_GOMOD_CACHE} -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		bash -c "git config --global --add safe.directory '*' && make build-test-so-local"

.PHONY: integration-test
integration-test:
	if ! docker images ${PROXY_IMAGE} | grep envoyproxy/envoy > /dev/null; then \
		docker pull ${PROXY_IMAGE}; \
	fi
	$(foreach PKG, $(shell go list ./tests/integration/...), \
		go test ${TEST_OPTION} ${PKG} || exit 1; \
	)

.PHONY: build-so-local
build-so-local:
	CGO_ENABLED=1 go build -tags so \
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X main.Version=${MAJOR_VERSION}(${GIT_VERSION})" \
		--buildmode=c-shared \
		-v -o ${TARGET_SO} \
		${PROJECT_NAME}/cmd/libgolang

.PHONY: build-so
build-so:
	docker run --rm ${MOUNT_GOMOD_CACHE} -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		bash -c "git config --global --add safe.directory '*' && make build-so-local"

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
	go run github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8 -w -local ${PROJECT_NAME} .

# Don't use `buf format` to format the protobuf files! Buf's code style is different from Envoy.
# That will break lots of things.
.PHONY: lint-proto
lint-proto:
	go run github.com/bufbuild/buf/cmd/buf@v1.28.1 lint

.PHONY: lint-spell
lint-spell: build-dev-tools
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		make lint-spell-local

.PHONY: lint-spell-local
lint-spell-local:
	codespell --skip '.git,.idea,test-envoy,go.mod,go.sum,*.svg' --check-filenames --check-hidden --ignore-words ./.ignore_words

.PHONY: lint-editorconfig
lint-editorconfig:
	go run github.com/editorconfig-checker/editorconfig-checker/cmd/editorconfig-checker@2.7.2

.PHONY: lint
lint: lint-go lint-proto lint-spell lint-editorconfig

.PHONY: fmt
fmt: fmt-go

.PHONY: verify-example
verify-example:
	cd ./examples/dev_your_plugin && ./verify.sh
