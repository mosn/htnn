#!/usr/bin/env bash
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


set -eo pipefail
set -x

export PATH="./bin:$PATH"

if [ -z "$ISTIO_NAMESPACE" ]; then
  export ISTIO_NAMESPACE="istio-system"
fi

install() {
    if [[ ! -x "$(command -v istioctl)" ]] || ! istioctl version 2>/dev/null | grep -q "$ISTIO_VERSION"; then
        echo "istioctl not found, installing..."
        curl -sL https://istio.io/downloadIstioctl | sh -
        mv "$HOME"/.istioctl/bin/istioctl ./bin
    fi
    istioctl version
    istioctl install --set profile=default -y
    istioctl manifest apply  --set .values.global.proxy.image="htnn/e2e-dp:0.1.0" -y
}

uninstall() {
    istioctl uninstall --purge -y
}

opt=$1
shift

${opt} "$@"
