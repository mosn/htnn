---
title: Installation
---

## Install via Helm

### Prerequisites

* Helm 3.6 or higher. For installing Helm, refer to the [Helm installation guide](https://helm.sh/docs/intro/install/).
* Configure helm repository address. Execute the following command to add the repository:

```shell
helm repo add mosn xxxx # TODO: setup such a repo
helm repo update
```

### Installation

Installation command:

```shell
$ helm install $package_name mosn/$package_name --namespace istio-system --create-namespace --wait
```

Where `$package_name` can be:

* `htnn-controller`: control plane component
* `htnn-gateway`: data plane component

### Configuration

We can use Helm's [Value files](https://helm.sh/docs/chart_template_guide/values_files/) to configure the default values of the Helm Chart. For example:

```shell
helm install htnn-controller mosn/htnn-controller ... --set istiod.pilot.env.HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS=true
```

Configurations related to the control plane start with `istiod`, for specific configuration items please refer to the [control plane configuration](https://github.com/istio/istio/blob/1.21.2/manifests/charts/istio-control/istio-discovery/values.yaml).

```shell
helm install htnn-gateway mosn/htnn-gateway ... --set gateway.podAnnotations.test=ok
```

Configurations related to the data plane start with `gateway`, for specific configuration items please refer to the [data plane configuration](https://github.com/istio/istio/blob/1.21.2/manifests/charts/gateway/values.yaml).
