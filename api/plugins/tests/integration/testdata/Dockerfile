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

# Dockerfile has specific requirement to put this ARG at the beginning:
# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG GO_BUILD_BASE_IMAGE
ARG PROXY_BASE_IMAGE

# hadolint ignore=DL3006
FROM ${PROXY_BASE_IMAGE} as envoy
# hadolint ignore=DL3006
FROM ${GO_BUILD_BASE_IMAGE}
COPY --from=envoy /usr/local/bin/envoy /usr/local/bin/envoy
ENTRYPOINT ["bash"]
