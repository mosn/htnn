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
	$(CONTROLLER_GEN) rbac:roleName=htnn-role crd:ignoreUnexportedFields=true \
		output:crd:artifacts:config=../manifests/charts/htnn-controller/templates/crds/ \
		output:rbac:artifacts:config=../manifests/charts/htnn-controller/templates/rbac/ \
		paths="../types/..." paths="./internal/controller/..."

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="../types/hack/boilerplate.go.txt" paths="../types/..."

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
test: manifests generate envtest install-crd-deps ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -gcflags="all=-N -l" -race \
		./... -coverprofile cover.out -covermode=atomic -coverpkg=${PROJECT_NAME}/...

.PHONY: benchmark
benchmark: manifests generate envtest install-crd-deps  ## Run benchmarks
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -tags benchmark -timeout=1200s -v ./tests/benchmark/...

##@ Deployment

ignore-not-found ?= false
KIND ?= kind

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
# Remember to remove tools downloaded into bin directory manually before updating them.
# If they need to be updated frequently, we can consider to store them in the `Dockerfile.dev`.
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.2.1
CONTROLLER_TOOLS_VERSION ?= v0.13.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: start-controller-service
start-controller-service:
	cd ./tests/testdata/services && docker-compose up -d

.PHONY: stop-controller-service
stop-controller-service:
	cd ./tests/testdata/services && docker-compose down
