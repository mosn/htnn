# htnn-gateway

![Version: 0.4.1](https://img.shields.io/badge/Version-0.4.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.4.1](https://img.shields.io/badge/AppVersion-0.4.1-informational?style=flat-square)

A Helm chart for HTNN data plane running as gateway

## Install

To install the chart with the release `htnn-gateway`:

```shell
helm repo add htnn https://mosn.github.io/htnn
helm repo update

helm install htnn-gateway htnn/htnn-gateway --namespace istio-system --create-namespace
```

For more information like how to configure and troubleshoot, please refer to the [Installation Guide](https://github.com/mosn/htnn/blob/main/site/content/en/docs/getting-started/installation.md).

### `image: auto` Information

The image used by the chart, `auto`, may be unintuitive.
This exists because the pod spec will be automatically populated at runtime, using the same mechanism as [Sidecar Injection](istio.io/latest/docs/setup/additional-setup/sidecar-injection).
This allows the same configurations and lifecycle to apply to gateways as sidecars.

Note: this does mean two things:

1. the namespace the gateway is deployed in must not have the `istio-injection=disabled` label.
See [Controlling the injection policy](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/#controlling-the-injection-policy) for more info.
2. the gateway must be installed after `htnn/htnn-controller` is installed so that the pod spec can be injected.

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
| https://istio-release.storage.googleapis.com/charts | gateway | 1.21.3 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| gateway.env.ISTIO_DELTA_XDS | string | `"true"` |  |
| gateway.name | string | `"istio-ingressgateway"` |  |

