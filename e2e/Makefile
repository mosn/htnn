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

include ../etc/common.mk

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
	cd ../controller/ && make docker-build

e2e-build-so:
	cd ../plugins/ && make build-so && mv libgolang.so ../e2e/

.PHONY: e2e-prepare-data-plane-image
e2e-prepare-data-plane-image: e2e-build-so kind
	docker build -t htnn/e2e-dp:0.1.0 .
	$(KIND) load docker-image htnn/e2e-dp:0.1.0 --name htnn

.PHONY: deploy-istio
deploy-istio:
	ISTIO_VERSION=1.21.0 ./istio.sh install
	$(KUBECTL) wait --timeout=5m -n istio-system deployment/istio-ingressgateway --for=condition=Available

.PHONY: deploy-cert-manager
deploy-cert-manager:
	$(KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
	$(KUBECTL) wait --timeout=5m -n cert-manager deployment/cert-manager-cainjector --for=condition=Available

.PHONY: deploy-controller
deploy-controller: kind kubectl
	cd ../controller/ && KIND=$(KIND) KIND_OPTION="-n htnn" KUBECTL=$(KUBECTL) make deploy
	$(KUBECTL) wait --timeout=5m -n controller-system deployment/controller-controller-manager --for=condition=Available

.PHONY: undeploy-controller
undeploy-controller: kubectl
	cd ../controller/ && KUBECTL=$(KUBECTL) make undeploy

.PHONY: run-e2e
run-e2e:
	PATH=$(LOCALBIN):"$(PATH)" go test -v ./...

# Run `make undeploy-controller e2e-build-controller-image  deploy-controller` to update the controller.
# To update the data plane, run `make e2e-prepare-data-plane-image` to update the image and then delete
# the ingressgateway pod to trigger restart.
.PHONY: e2e-ci
e2e-ci: delete-cluster create-cluster deploy-cert-manager e2e-prepare-data-plane-image deploy-istio \
	e2e-build-controller-image deploy-controller run-e2e
