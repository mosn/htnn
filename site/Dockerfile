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

# Use debian image for ARM64 support
FROM floryn90/hugo:ext-debian@sha256:55e887d8847bb430b11360e430475dd9b7a7d968aa6e5b4d4440c91f3f64d894

RUN apt install -y --no-install-recommends git && \
  git config --global --add safe.directory /src
