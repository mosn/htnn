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

include common.mk

# For some tools, like golangci-lint, we prefer to use the latest version so that we can have the new feature.
# For the other tools, like kind, we don't upgrade it until there is a strong reason.

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

GO_FMTTER_VERSION = 0.3.8
.PHONY: install-go-fmtter
install-go-fmtter: $(LOCALBIN)
	if ! test -x $(LOCALBIN)/gosimports || ! $(LOCALBIN)/gosimports -version | grep $(GO_FMTTER_VERSION) >/dev/null; then \
		GOBIN=$(LOCALBIN) go install github.com/rinchsan/gosimports/cmd/gosimports@v$(GO_FMTTER_VERSION); \
	fi

.PHONY: gen-proto
gen-proto: dev-tools install-go-fmtter $(GO_TARGETS)
# format the generated Go code so the `fmt-go` task can pass
%.pb.go: %.proto
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} --user $(shell id -u) -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		protoc --proto_path=. --go_opt="paths=source_relative" --go_out=. --validate_out="lang=go,paths=source_relative:." \
			-I /go/src/protoc-gen-validate $<
	$(LOCALBIN)/gosimports -w -local ${PROJECT_NAME} $@

.PHONY: gen-crd-code
gen-crd-code: $(LOCALBIN) install-go-fmtter
	test -s $(LOCALBIN)/client-gen || GOBIN=$(LOCALBIN) go install k8s.io/code-generator/cmd/client-gen@v0.29.3
	LOCALBIN=$(LOCALBIN) tools/gen-crd-code.sh
	$(LOCALBIN)/gosimports -w -local ${PROJECT_NAME} ./types/pkg/client

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

GOLANGCI_LINT_VERSION = 1.56.1
.PHONY: lint-go
lint-go:
	if ! test -x $(LOCALBIN)/golangci-lint || ! $(LOCALBIN)/golangci-lint --version | grep $(GOLANGCI_LINT_VERSION) >/dev/null; then \
		GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION); \
	fi
	$(foreach PKG, $(GO_PROD_MODULES), \
		pushd ./${PKG} && $(LOCALBIN)/golangci-lint run --config=../.golangci.yml || exit 1; popd; \
	)

.PHONY: fmt-go
fmt-go: install-go-fmtter
	$(foreach PKG, $(GO_MODULES), \
		pushd ./${PKG} && \
			go mod tidy || exit 1; \
			$(LOCALBIN)/gosimports -w -local ${PROJECT_NAME} . || exit 1; \
		popd; \
	)

# Don't use `buf format` to format the protobuf files! Buf's code style is different from Envoy.
# That will break lots of things.
.PHONY: lint-proto
lint-proto: $(LOCALBIN)
	test -x $(LOCALBIN)/buf || GOBIN=$(LOCALBIN) go install github.com/bufbuild/buf/cmd/buf@v1.28.1
	$(LOCALBIN)/buf lint --exclude-path ./external

.PHONY: fmt-proto
fmt-proto: dev-tools
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		make fmt-proto-local

.PHONY: fmt-proto-local
fmt-proto-local:
	find . -name '*.proto' | grep -v './external' | xargs clang-format -i


LICENSE_CHECKER_VERSION = 0.6.0
.PHONY: install-license-checker
install-license-checker: $(LOCALBIN)
	if ! test -x $(LOCALBIN)/license-eye || ! $(LOCALBIN)/license-eye --version | grep $(LICENSE_CHECKER_VERSION) >/dev/null; then \
		GOBIN=$(LOCALBIN) go install github.com/apache/skywalking-eyes/cmd/license-eye@v$(LICENSE_CHECKER_VERSION); \
	fi

.PHONY: lint-license
lint-license: install-license-checker
	$(LOCALBIN)/license-eye header check
	$(LOCALBIN)/license-eye dependency check

.PHONY: fix-license
fix-license: install-license-checker
	$(LOCALBIN)/license-eye header fix

.PHONY: lint-spell
lint-spell: dev-tools
	docker run --rm -v $(PWD):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} \
		${DEV_TOOLS_IMAGE} \
		make lint-spell-local

CODESPELL = codespell --skip 'test-envoy,go.mod,go.sum,*.patch,*.svg,./site/public/**' --check-filenames --check-hidden --ignore-words ./.ignore_words $(shell ls -A | tr '\t' '\n' | grep -vE 'external|.git|.idea|go.work.sum')
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

TARGET_ISTIO_DIR = $(shell pwd)/external/istio

.PHONY: prebuild
prebuild:
	git submodule update --init --recursive
	cd $(TARGET_ISTIO_DIR) && git status | grep -q "nothing to commit, working tree clean" || (echo "istio submodule is not clean, please commit your changes first"; exit 1)
	cd ./patch && ./apply-patch.sh $(TARGET_ISTIO_DIR)
