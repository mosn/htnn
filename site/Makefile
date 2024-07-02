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

.PHONY: build
build:
	docker-compose build

.PHONY: up
up:
	docker-compose up

.PHONY: clean
clean:
	docker-compose rm

.PHONY: publish
# The generated files will be under ./public
publish:
	# FIXME: change baseURL to real link
	docker run --rm -v $(PWD):/src --name docsy htnn-docsy -- hugo --minify --gc --baseURL https://mosn.io/

.PHONY: lint-website
lint-website: $(LOCALBIN)
	test -x $(LOCALBIN)/htmltest || GOBIN=$(LOCALBIN) go install github.com/wjdp/htmltest@v0.17.0
	$(LOCALBIN)/htmltest --conf ./.htmltest.yml ./public | grep 'target does not exist' && exit 1 || true
