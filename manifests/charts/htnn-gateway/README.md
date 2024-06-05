# htnn-gateway

![Version: 0.1.1](https://img.shields.io/badge/Version-0.1.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.2.1](https://img.shields.io/badge/AppVersion-0.2.1-informational?style=flat-square)

A Helm chart for HTNN data plane running as gateway

## Install

To install the chart with the release `htnn-gateway`:

```shell
helm repo add htnn https://mosn.github.io/htnn
helm repo update

helm install htnn-gateway htnn/htnn-gateway --namespace istio-system --create-namespace --wait --debug
```

## Uninstall

To uninstall the Helm release `htnn-gateway`:

```shell
helm uninstall htnn-gateway -n istio-system
```

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| spacewander |  |  |

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://istio-release.storage.googleapis.com/charts | gateway | 1.21.2 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| gateway.env.ISTIO_DELTA_XDS | string | `"true"` |  |
| gateway.name | string | `"istio-ingressgateway"` |  |

