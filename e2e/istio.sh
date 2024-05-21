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

ISTIOCTL=./bin/istioctl

install() {
    if ! $ISTIOCTL version 2>/dev/null | grep -q "$ISTIO_VERSION"; then
        echo "matched istioctl not found, installing..."
        curl -sL https://istio.io/downloadIstioctl | sh -
        mv "$HOME"/.istioctl/bin/istioctl ./bin
    fi
    $ISTIOCTL install --set profile=default -y
    $ISTIOCTL version
    # the image name should be in ns/name format, otherwise istio will add ":ver" suffix to it
    $ISTIOCTL manifest apply \
        --set .values.pilot.image="htnn/e2e-cp:0.1.0" \
        --set .values.pilot.env.PILOT_SCOPE_GATEWAY_TO_NAMESPACE=true \
        --set .values.pilot.env.PILOT_ENABLE_HTNN=true \
        --set .values.pilot.env.PILOT_ENABLE_HTNN_STATUS=true \
        --set .values.pilot.env.HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS=true \
        --set .values.pilot.env.UNSAFE_PILOT_ENABLE_RUNTIME_ASSERTIONS=true \
        --set .values.pilot.env.UNSAFE_PILOT_ENABLE_DELTA_TEST=true \
        --set .values.global.proxy.image="htnn/e2e-dp:0.1.0" \
        --set .values.global.imagePullPolicy=Never \
        --set .values.global.logging.level=default:info,htnn:debug \
        --set meshConfig.defaultConfig.proxyMetadata.ISTIO_DELTA_XDS=true \
        -y || exitWithAnalysis
}

exitWithAnalysis() {
    kubectl get pods -n istio-system -o yaml
    for pod in $(kubectl get pods -n istio-system | grep 'istiod-' | awk '{print $1}'); do
        kubectl -n istio-system logs --tail=1000 "$pod"
        echo
    done
    for pod in $(kubectl get pods -n istio-system | grep 'istio-ingressgateway' | awk '{print $1}'); do
        kubectl -n istio-system logs --tail=1000 "$pod"
        echo
    done
    exit 1
}

uninstall() {
    $ISTIOCTL uninstall --purge -y
}

opt=$1
shift

${opt} "$@"
