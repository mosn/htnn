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
FROM floryn90/hugo:ext-debian@sha256:150de43ac33d67f6681914d91c8520eb3438d93258fcfbf8e9534cd0e5031142

RUN apt install -y --no-install-recommends git && \
  git config --global --add safe.directory /src
