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

.PHONY: unit-test
unit-test:
	go test -tags envoy${ENVOY_API_VERSION} ${TEST_OPTION} $(shell go list ./... | \
		grep -v tests/integration)

# We can't specify -race to `go build` because it seems that
# race detector assumes that the executable is loaded around the 0 address. When loaded by the Envoy,
# the race detector will allocate memory out of 48bits address which is not allowed in x64.
.PHONY: build-test-so-local
build-test-so-local:
	CGO_ENABLED=1 go build -tags so,envoy${ENVOY_API_VERSION} \
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') " \
		--buildmode=c-shared \
		-cover -covermode=atomic -coverpkg=${PROJECT_NAME}/... \
		-v -o tests/integration/${TARGET_SO} \
		${PROJECT_NAME}/api/tests/integration/libgolang

# Go 1.19+ adds vcs check which will cause error "fatal: detected dubious ownership in repository at '...'".
# So here we disable the error via git configuration when running inside Docker.
.PHONY: build-test-so
build-test-so:
	docker run --rm ${MOUNT_GOMOD_CACHE} \
		-v $(PWD)/..:/go/src/${PROJECT_NAME} \
		-w /go/src/${PROJECT_NAME}/api \
		-e GOPROXY \
		-e ENVOY_API_VERSION \
		${BUILD_IMAGE} \
		bash -c "git config --global --add safe.directory '*' && make build-test-so-local"

# The data plane image used in the integration test can be controlled via env var PROXY_IMAGE
.PHONY: integration-test
integration-test:
	if ! command -v grpcurl >/dev/null 2>&1; then go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest; fi
	test -d /tmp/htnn_coverage && rm -rf /tmp/htnn_coverage || true
	$(foreach PKG, $(shell go list ./tests/integration/...), \
		go test -tags envoy${ENVOY_API_VERSION} -v ${PKG} || exit 1; \
	)

.PHONY: test-integration-framework-in-docker
test-integration-framework-in-docker:
	docker build -t test-integration-framework-in-docker \
		--build-arg PROXY_BASE_IMAGE=${PROXY_IMAGE} \
		--build-arg GO_BUILD_BASE_IMAGE=${BUILD_IMAGE} \
		plugins/tests/integration/testdata
	docker run --rm ${MOUNT_GOMOD_CACHE} \
		-v $(PWD)/..:/go/src/${PROJECT_NAME} \
		-w /go/src/${PROJECT_NAME}/api \
		-e GOPROXY \
		-e ENVOY_API_VERSION \
		test-integration-framework-in-docker \
		/go/src/${PROJECT_NAME}/api/plugins/tests/integration/test_binary_mode.sh

# The benchmark running time can be controlled via env var HTNN_DATA_PLANE_BENCHMARK_DURATION
.PHONY: benchmark
benchmark:
	go test -tags benchmark,envoy${ENVOY_API_VERSION} -v ./tests/integration/ -run TestBenchmark

.PHONY: start-service
start-service:
	cd ./tests/integration/testdata/services && docker compose up -d --build

.PHONY: stop-service
stop-service:
	cd ./tests/integration/testdata/services && docker compose down
