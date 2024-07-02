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

KUBECTL ?= $(LOCALBIN)/kubectl
KIND ?= $(LOCALBIN)/kind
HELM ?= $(LOCALBIN)/helm

.PHONY: kubectl
kubectl: $(LOCALBIN)
	@test -x $(KUBECTL) || \
		KUBECTL_VERSION=v$(MIN_K8S_VERSION) LOCATION=$(KUBECTL) ./k8s.sh install-kubectl

.PHONY: kind
kind: $(LOCALBIN)
	@test -x $(KIND) || GOBIN=$(LOCALBIN) go install sigs.k8s.io/kind@v0.20.0

# we can't install it via `go install` because a limitation: https://github.com/golang/go/issues/44840
.PHONY: helm
helm: $(LOCALBIN)
	@test -x $(HELM) || curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | USE_SUDO=false HELM_INSTALL_DIR=$(LOCALBIN) bash

.PHONY: create-cluster
create-cluster: kind kubectl
	$(KIND) create cluster --name htnn --image kindest/node:v$(MIN_K8S_VERSION)
	$(KUBECTL) kustomize "github.com/kubernetes-sigs/gateway-api/config/crd?ref=v$(GATEWAY_API_VERSION)" | $(KUBECTL) apply -f -

.PHONY: delete-cluster
delete-cluster: kind
	$(KIND) delete cluster --name htnn || true

.PHONY: e2e-prepare-controller-image
e2e-prepare-controller-image: kind
	cd ../manifests/ && CONTROLLER_IMAGE=htnn/controller:e2e make build-controller-image
	$(KIND) load docker-image -n htnn htnn/controller:e2e

.PHONY: e2e-prepare-data-plane-image
e2e-prepare-data-plane-image: kind
	cd ../manifests/ && PROXY_IMAGE=htnn/gateway:e2e make build-proxy-image
	$(KIND) load docker-image htnn/gateway:e2e --name htnn

.PHONY: deploy-istio
deploy-istio: helm
	ISTIO_VERSION=$(ISTIO_VERSION) LOCALBIN=$(LOCALBIN) ./istio.sh install

.PHONY: run-e2e
run-e2e:
	PATH=$(LOCALBIN):"$(PATH)" go test -v ./... || (bash -x ./collect_e2e_logs.sh; exit -1)

# To update the control plane, run `make e2e-prepare-controller-image`, and then delete
# the istiod pod to trigger restart.
# To update the data plane, run `make e2e-prepare-data-plane-image` to update the image and then delete
# the data plane pod to trigger restart.
.PHONY: e2e-ci
e2e-ci: delete-cluster create-cluster e2e-prepare-data-plane-image \
	e2e-prepare-controller-image deploy-istio run-e2e
