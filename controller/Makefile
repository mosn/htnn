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

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.28.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GINKGO_OPTIONS ?=

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate ClusterRole and CustomResourceDefinition objects.
	@if test -s $(LOCALBIN)/controller-gen; then \
		$(CONTROLLER_GEN) rbac:roleName=htnn-role crd:ignoreUnexportedFields=true \
			output:crd:artifacts:config=../manifests/charts/htnn-controller/templates/crds/ \
			output:rbac:artifacts:config=../manifests/charts/htnn-controller/templates/rbac/ \
			paths="../types/..." paths="./internal/controller/..."; \
	else \
		echo "WARNING: controller-gen not available, skipping manifests generation"; \
	fi

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	@if test -s $(LOCALBIN)/controller-gen; then \
		$(CONTROLLER_GEN) object:headerFile="../types/hack/boilerplate.go.txt" paths="../types/..."; \
	else \
		echo "WARNING: controller-gen not available, skipping code generation"; \
	fi

.PHONY: install-crd-deps
install-crd-deps:
	test -d tests/testdata/crd || mkdir -p tests/testdata/crd
	test -f tests/testdata/crd/gateway-api-$(GATEWAY_API_VERSION).yaml || \
		(wget https://github.com/kubernetes-sigs/gateway-api/releases/download/v$(GATEWAY_API_VERSION)/standard-install.yaml -O \
			temp.out && mv temp.out tests/testdata/crd/gateway-api-$(GATEWAY_API_VERSION).yaml)
	test -f tests/testdata/crd/istio-$(ISTIO_VERSION).yaml || \
		(wget https://raw.githubusercontent.com/istio/istio/$(ISTIO_VERSION)/manifests/charts/base/crds/crd-all.gen.yaml -O \
			temp.out && mv temp.out tests/testdata/crd/istio-$(ISTIO_VERSION).yaml)

.PHONY: test
test: manifests generate envtest install-crd-deps
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ${TEST_OPTION} ./...

.PHONY: benchmark
benchmark: manifests generate envtest install-crd-deps
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -tags benchmark -timeout=1200s -v ./tests/benchmark/...

.PHONY: integration-test
integration-test: manifests generate envtest install-crd-deps
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -count 1 -v ${TEST_OPTION} ./tests/integration/... ${GINKGO_OPTIONS}

##@ Deployment

ignore-not-found ?= false
KIND ?= kind

##@ Build Dependencies

## Tool Binaries
KUBECTL ?= kubectl
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.2.1
CONTROLLER_TOOLS_VERSION ?= v0.16.5

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	(GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION) || \
	echo "WARNING: controller-gen $(CONTROLLER_TOOLS_VERSION) failed to install, skipping code generation")

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: start-service
start-service:
	cd ./tests/testdata/services && docker compose up -d
	@echo "=== Docker container status ==="
	cd ./tests/testdata/services && docker compose ps
	@echo "=== Waiting for primary Nacos (port 8848) to be ready..."
	@for i in $$(seq 1 90); do \
		if curl -sf http://0.0.0.0:8848/nacos/v1/ns/service/list?pageNo=1\&pageSize=1 >/dev/null 2>&1; then \
			echo "Primary Nacos is ready (after $${i}s)"; \
			break; \
		fi; \
		if [ $$i -eq 45 ]; then \
			echo "=== [45s] Nacos still not ready, checking container health ==="; \
			docker compose -f ./tests/testdata/services/docker-compose.yml ps nacos; \
			echo "=== nacos logs (last 50 lines) ==="; \
			docker compose -f ./tests/testdata/services/docker-compose.yml logs --no-log-prefix --tail=50 nacos; \
		fi; \
		if [ $$i -eq 90 ]; then \
			echo "ERROR: Primary Nacos not ready after 90s."; \
			echo "=== nacos container full logs ==="; \
			docker compose -f ./tests/testdata/services/docker-compose.yml logs --no-log-prefix nacos; \
			echo "=== nacos2 container full logs ==="; \
			docker compose -f ./tests/testdata/services/docker-compose.yml logs --no-log-prefix nacos2; \
			echo "=== docker system info ==="; \
			docker system df; \
		fi; \
		sleep 1; \
	done

.PHONY: stop-service
stop-service:
	cd ./tests/testdata/services && docker compose down
