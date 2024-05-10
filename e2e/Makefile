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

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

KUBECTL ?= $(LOCALBIN)/kubectl
KIND ?= $(LOCALBIN)/kind

.PHONY: kubectl
kubectl: $(LOCALBIN)
	@test -x $(KUBECTL) || \
		KUBECTL_VERSION=v$(MIN_K8S_VERSION) LOCATION=$(KUBECTL) ./k8s.sh install-kubectl

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
	cd ../controller/ && IMG=htnn/e2e-cp:0.1.0 CONTROLLER_BASE_IMAGE=docker.io/istio/pilot:$(ISTIO_VERSION) make docker-build

e2e-build-so:
	cd ../plugins/ && make build-so && mv libgolang.so ../e2e/

.PHONY: e2e-prepare-data-plane-image
e2e-prepare-data-plane-image: e2e-build-so kind
	cd .. && docker build -t htnn/e2e-dp:0.1.0 --build-arg GOPROXY=${GOPROXY} -f e2e/Dockerfile .
	$(KIND) load docker-image htnn/e2e-dp:0.1.0 --name htnn

.PHONY: deploy-istio
deploy-istio:
	$(KUBECTL) apply -f ./base/istio_prestart.yaml
	ISTIO_VERSION=$(ISTIO_VERSION) ./istio.sh install

.PHONY: deploy-controller
deploy-controller: kind kubectl
	cd ../controller/ && KIND=$(KIND) KIND_OPTION="-n htnn" KUBECTL=$(KUBECTL) IMG=htnn/e2e-cp:0.1.0 make deploy

.PHONY: undeploy-controller
undeploy-controller: kubectl
	cd ../controller/ && KUBECTL=$(KUBECTL) make undeploy

.PHONY: run-e2e
run-e2e:
	PATH=$(LOCALBIN):"$(PATH)" go test -v ./... || (bash -x ./collect_e2e_logs.sh; exit -1)

# To update the control plane, run `make e2e-prepare-controller-image deploy-controller`, and the delete
# the istiod pod to trigger restart.
# To update the data plane, run `make e2e-prepare-data-plane-image` to update the image and then delete
# the data plane pod to trigger restart.
.PHONY: e2e-ci
e2e-ci: delete-cluster create-cluster e2e-prepare-data-plane-image \
	e2e-build-controller-image deploy-controller deploy-istio run-e2e
