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
OS = $(shell uname)
IN_CI ?=

TARGET_SO       = libgolang.so
PROJECT_NAME    = mosn.io/htnn
# Both images use glibc 2.31. Ensure libc in the images match each other.
BUILD_IMAGE     ?= golang:1.21-bullseye
# This is the envoyproxy/envoy:contrib-v1.29-latest
# Use docker inspect --format='{{index .RepoDigests 0}}' envoyproxy/envoy:contrib-v1.29-latest
# to get the sha256 ID
PROXY_IMAGE     ?= envoyproxy/envoy@sha256:98ed3d86ff8b86dc12ddf54b7bb67ddf5506f80769038b3e2ab7bf402730fb4d
# We may need to use timestamp if we need to update the image in one PR
DEV_TOOLS_IMAGE ?= ghcr.io/mosn/htnn-dev-tools:2024-01-18

VERSION   = $(shell cat VERSION)
GIT_VERSION     = $(shell git log -1 --pretty=format:%h)

MIN_K8S_VERSION = 1.25.11

# Define a recursive wildcard function
rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$d/,$(2))$(filter $(subst *,%,$(2)),$d))

PROTOC = protoc
PROTO_FILES = $(call rwildcard,./,*.proto)
GO_TARGETS = $(patsubst %.proto,%.pb.go,$(PROTO_FILES))

TEST_OPTION ?= -gcflags="all=-N -l" -race

MOUNT_GOMOD_CACHE = -v $(shell go env GOPATH):/go
ifeq ($(IN_CI), true)
	# Mount go mod cache in the CI environment will cause 'Permission denied' error
	# when accessing files on host in later phase because the mounted directory will
	# have files which is created by the root user in Docker.
	# Run as low privilege user in the Docker doesn't
	# work because we also need root to create /.cache in the Docker.
	MOUNT_GOMOD_CACHE =
endif

LOCALBIN ?= $(shell pwd)/bin
# Remember to remove tools downloaded into bin directory manually before updating them.
# If they need to be updated frequently, we can consider to store them in the `Dockerfile.dev`.
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

.PHONY: install-go-fmtter
install-go-fmtter: $(LOCALBIN)
	test -x $(LOCALBIN)/gosimports || GOBIN=$(LOCALBIN) go install github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8

.PHONY: gen-proto
gen-proto: dev-tools install-go-fmtter $(GO_TARGETS)
# format the generated Go code so the `fmt-go` task can pass
%.pb.go: %.proto
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} --user $(shell id -u) -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		protoc --proto_path=. --go_opt="paths=source_relative" --go_out=. --validate_out="lang=go,paths=source_relative:." \
			-I ../../protoc-gen-validate $<
	$(LOCALBIN)/gosimports -w -local ${PROJECT_NAME} $@

# We don't run the controller's unit test in this task. Because the controller is considered as a
# separate component.
.PHONY: unit-test
unit-test:
	go test ${TEST_OPTION} $(shell go list ./... | \
		grep -v tests/integration | \
		grep -v ${PROJECT_NAME}/controller | \
		grep -v ${PROJECT_NAME}/e2e)

# We can't specify -race to `go build` because it seems that
# race detector assumes that the executable is loaded around the 0 address. When loaded by the Envoy,
# the race detector will allocate memory out of 48bits address which is not allowed in x64.
.PHONY: build-test-so-local
build-test-so-local:
	CGO_ENABLED=1 go build -tags so \
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X main.Version=${VERSION}(${GIT_VERSION})" \
		--buildmode=c-shared \
		-v -o plugins/tests/integration/${TARGET_SO} \
		${PROJECT_NAME}/plugins/tests/integration/libgolang

# Go 1.19+ adds vcs check which will cause error "fatal: detected dubious ownership in repository at '...'".
# So here we disable the error via git configuration when running inside Docker.
.PHONY: build-test-so
build-test-so:
	docker run --rm ${MOUNT_GOMOD_CACHE} -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		-e GOPROXY \
		${BUILD_IMAGE} \
		bash -c "git config --global --add safe.directory '*' && make build-test-so-local"

.PHONY: plugins-integration-test
plugins-integration-test:
	if ! docker images ${PROXY_IMAGE} | grep envoyproxy/envoy > /dev/null; then \
		docker pull ${PROXY_IMAGE}; \
	fi
	$(foreach PKG, $(shell go list ./plugins/tests/integration/...), \
		go test -v ${PKG} || exit 1; \
	)

.PHONY: build-so-local
build-so-local:
	CGO_ENABLED=1 go build -tags so \
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X main.Version=${VERSION}(${GIT_VERSION})" \
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

.PHONY: dev-tools
dev-tools:
	@if ! docker images ${DEV_TOOLS_IMAGE} | grep dev-tools > /dev/null; then \
		docker pull ${DEV_TOOLS_IMAGE}; \
	fi

# `--network=host` is used to access GitHub. You might need to configure `docker buildx` to enable it.
# See https://github.com/docker/buildx/issues/835#issuecomment-966496802
.PHONY: build-dev-tools
build-dev-tools:
	# before running this task, please run `make build-dev-tools-local` and check if the image work locally
	docker buildx build --platform=linux/amd64,linux/arm64 \
		--network=host --build-arg GOPROXY=${GOPROXY} -t ${DEV_TOOLS_IMAGE} --push -f tools/Dockerfile.dev ./tools

.PHONY: build-dev-tools-local
build-dev-tools-local:
	docker build --network=host --build-arg GOPROXY=${GOPROXY} -t ${DEV_TOOLS_IMAGE} -f tools/Dockerfile.dev ./tools

# For lint-go/fmt-go: we don't cover examples/dev_your_plugin which is just an example

.PHONY: lint-go
lint-go:
	test -x $(LOCALBIN)/golangci-lint || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2
	$(LOCALBIN)/golangci-lint run --config=.golangci.yml

.PHONY: fmt-go
fmt-go: install-go-fmtter
	go mod tidy
	$(LOCALBIN)/gosimports -w -local ${PROJECT_NAME} .

# Don't use `buf format` to format the protobuf files! Buf's code style is different from Envoy.
# That will break lots of things.
.PHONY: lint-proto
lint-proto: $(LOCALBIN)
	test -x $(LOCALBIN)/buf || GOBIN=$(LOCALBIN) go install github.com/bufbuild/buf/cmd/buf@v1.28.1
	$(LOCALBIN)/buf lint

.PHONY: fmt-proto
fmt-proto: dev-tools
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		make fmt-proto-local

.PHONY: fmt-proto-local
fmt-proto-local:
	find . -name '*.proto' -exec clang-format -i {} \+

.PHONY: install-license-checker
install-license-checker: $(LOCALBIN)
	test -x $(LOCALBIN)/license-eye || GOBIN=$(LOCALBIN) go install github.com/apache/skywalking-eyes/cmd/license-eye@v0.5.0

.PHONY: lint-license
lint-license: install-license-checker
	$(LOCALBIN)/license-eye header check

.PHONY: fix-license
fix-license: install-license-checker
	$(LOCALBIN)/license-eye header fix

.PHONY: lint-spell
lint-spell: dev-tools
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		make lint-spell-local

CODESPELL = codespell --skip '.git,.idea,test-envoy,go.mod,go.sum,*.svg,./site/public/**' --check-filenames --check-hidden --ignore-words ./.ignore_words
.PHONY: lint-spell-local
lint-spell-local:
	$(CODESPELL)

.PHONY: fix-spell
fix-spell: dev-tools
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		make fix-spell-local

.PHONY: fix-spell-local
fix-spell-local:
	$(CODESPELL) -w

.PHONY: lint-editorconfig
lint-editorconfig: $(LOCALBIN)
	test -x $(LOCALBIN)/editorconfig-checker || GOBIN=$(LOCALBIN) go install github.com/editorconfig-checker/editorconfig-checker/cmd/editorconfig-checker@2.7.2
	$(LOCALBIN)/editorconfig-checker


.PHONY: lint-remain
lint-remain:
	grep '>>>>>>' $(shell git ls-files .) | grep -v 'Makefile:' && exit 1 || true
	go run tools/cmd/linter/main.go

.PHONY: lint
lint: lint-go lint-proto lint-license lint-spell lint-editorconfig lint-remain

.PHONY: fmt
fmt: fmt-go fmt-proto

.PHONY: verify-example
verify-example:
	cd ./examples/dev_your_plugin && ./verify.sh

.PHONY: start-service
start-service:
	cd ./ci/ && docker-compose up -d

# E2E
KUBECTL ?= $(LOCALBIN)/kubectl
KIND ?= $(LOCALBIN)/kind

.PHONY: kubectl
kubectl: $(LOCALBIN)
	@test -x $(KUBECTL) || \
		KUBECTL_VERSION=v$(MIN_K8S_VERSION) LOCATION=$(KUBECTL) ./e2e/k8s.sh install-kubectl

.PHONY: kind
kind: $(LOCALBIN)
	@test -x $(KIND) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/kind@v0.20.0

.PHONY: create-cluster
create-cluster: kind kubectl
	$(KIND) create cluster --name htnn --image kindest/node:v$(MIN_K8S_VERSION)
	$(KUBECTL) kustomize "github.com/kubernetes-sigs/gateway-api/config/crd?ref=v1.0.0" | $(KUBECTL) apply -f -

.PHONY: delete-cluster
delete-cluster: kind
	$(KIND) delete cluster --name htnn || true

.PHONY: e2e-build-controller-image
e2e-build-controller-image:
	cd controller/ && make docker-build

.PHONY: e2e-prepare-data-plane-image
e2e-prepare-data-plane-image: build-so kind
	docker build -t htnn/e2e-dp:0.1.0 -f e2e/Dockerfile .
	$(KIND) load docker-image htnn/e2e-dp:0.1.0 --name htnn

.PHONY: deploy-istio
deploy-istio:
	ISTIO_VERSION=1.20.0 ./e2e/istio.sh install
	$(KUBECTL) wait --timeout=5m -n istio-system deployment/istio-ingressgateway --for=condition=Available

.PHONY: deploy-cert-manager
deploy-cert-manager:
	$(KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
	$(KUBECTL) wait --timeout=5m -n cert-manager deployment/cert-manager-cainjector --for=condition=Available

.PHONY: deploy-controller
deploy-controller: kubectl
	cd controller/ && KIND=$(KIND) KIND_OPTION="-n htnn" KUBECTL=$(KUBECTL) make deploy
	$(KUBECTL) wait --timeout=5m -n controller-system deployment/controller-controller-manager --for=condition=Available

.PHONY: undeploy-controller
undeploy-controller: kubectl
	cd controller/ && KUBECTL=$(KUBECTL) make undeploy

.PHONY: run-e2e
run-e2e:
	PATH=$(LOCALBIN):"$(PATH)" go test -v ./e2e

# Run `make undeploy-controller e2e-build-controller-image  deploy-controller` to update the controller.
# To update the data plane, run `make e2e-prepare-data-plane-image` to update the image and then delete
# the ingressgateway pod to trigger restart.
.PHONY: e2e-ci
e2e-ci: delete-cluster create-cluster deploy-cert-manager e2e-prepare-data-plane-image deploy-istio e2e-build-controller-image deploy-controller run-e2e
