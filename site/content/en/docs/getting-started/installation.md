---
title: Installation
---

## Install via Helm

### Prerequisites

* Helm 3.6 or higher. For installing Helm, refer to the [Helm installation guide](https://helm.sh/docs/intro/install/).
* Configure helm repository address. Execute the following command to add the repository:

```shell
helm repo add htnn https://mosn.github.io/htnn
helm repo update
```

### Installation

Installation command:

```shell
helm install $package_name htnn/$package_name --namespace istio-system --create-namespace --wait
```

Where `$package_name` can be:

* `htnn-controller`: control plane component
* `htnn-gateway`: data plane component

Note that the pod spec of `htnn-gateway` will be automatically populated at runtime, using the same mechanism as [Sidecar Injection](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection).

This does mean two things:

1. the namespace the `htnn-gateway` is deployed in must not have the `istio-injection=disabled` label.
See [Controlling the injection policy](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/#controlling-the-injection-policy) for more info.
2. the `htnn-gateway` must be installed after `htnn-controller` is installed so that the pod spec can be injected.

If you set up the k8s environment with `kind`, the `helm install ... htnn/htnn-gateway --wait` will fail because `kind` doesn't support LoadBalancer service by default. You can use `kubectl wait --timeout=5m -n istio-system deployment/istio-ingressgateway --for=condition=Available` to indicate if the installation is finished.

### Configuration

We can use Helm's [Value files](https://helm.sh/docs/chart_template_guide/values_files/) to configure the default values of the Helm Chart. For example:

```shell
helm install htnn-controller htnn/htnn-controller ... --set pilot.env.HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS=true
```

For specific configuration items of htnn-controller, please refer to the [htnn-controller configuration](https://artifacthub.io/packages/helm/htnn/htnn-controller#values).

```shell
helm install htnn-gateway htnn/htnn-gateway ... --set gateway.podAnnotations.test=ok
```

Configurations related to the htnn-gateway start with `gateway`, for specific configuration items please refer to the [data plane configuration](https://github.com/istio/istio/blob/1.21.3/manifests/charts/gateway/values.yaml).
