# htnn-controller

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.2.1](https://img.shields.io/badge/AppVersion-0.2.1-informational?style=flat-square)

A Helm chart for HTNN controller

## Install

To install the chart with the release `htnn-controller`:

```shell
helm repo add mosn xxx # TODO: given the real one
helm repo update

helm install htnn-controller mosn/htnn-controller --namespace istio-system --create-namespace --wait --debug
```

## Uninstall

To uninstall the Helm release `htnn-controller`:

```shell
helm uninstall htnn-controller -n istio-system
```

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| spacewander |  |  |

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://istio-release.storage.googleapis.com/charts | istio-base(base) | 1.21.2 |
| https://istio-release.storage.googleapis.com/charts | istiod | 1.21.2 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| istiod.global.istioNamespace | string | `"istio-system"` |  |
| istiod.global.proxy.image | string | `"ghcr.io/mosn/htnn-proxy:dev"` |  |
| istiod.pilot.env.HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS | string | `"false"` |  |
| istiod.pilot.env.PILOT_ENABLE_HTNN | string | `"true"` |  |
| istiod.pilot.env.PILOT_ENABLE_HTNN_STATUS | string | `"true"` |  |
| istiod.pilot.env.PILOT_SCOPE_GATEWAY_TO_NAMESPACE | string | `"true"` |  |
| istiod.pilot.image | string | `"ghcr.io/mosn/htnn-controller:dev"` |  |
| istiod.revision | string | `""` |  |

