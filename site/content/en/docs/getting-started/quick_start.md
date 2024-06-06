---
title: Quick Start
---

## Prerequisites

* Kubernetes 1.26.0 or higher version. Using [kind](https://kind.sigs.k8s.io/) is recommended for quickly setting up a Kubernetes cluster locally.
* Helm 3.6 or higher version. For installing Helm, refer to the [Helm installation guide](https://helm.sh/docs/intro/install/).
* Configure helm repository address. Execute the following commands to add the repository:

```shell
helm repo add htnn https://mosn.github.io/htnn
helm repo update
```

## Installing HTNN

1. Install the control plane component:

```shell
$ helm install htnn-controller htnn/htnn-controller --namespace istio-system --create-namespace --debug --wait
NAME: htnn-controller
LAST DEPLOYED: Wed May 29 18:42:18 2024
NAMESPACE: istio-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

2. Install the data plane component:

```shell
$ helm install htnn-gateway htnn/htnn-gateway --namespace istio-system --create-namespace && \
    kubectl wait --timeout=5m -n istio-system deployment/istio-ingressgateway --for=condition=Available
NAME: htnn-gateway
LAST DEPLOYED: Wed May 29 19:59:22 2024
NAMESPACE: istio-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

Here we have not used the `--wait` parameter, instead, we used the `kubectl wait` command to wait for the `istio-ingressgateway` deployment to complete. Because `kind` does not support LoadBalancer type of Service by default, the ExternalIP for Service `istio-ingressgateway` will remain in `Pending` status. This does not affect our hands-on experience. If you're interested in this, refer to the [kind official documentation](https://kind.sigs.k8s.io/docs/user/loadbalancer/) and consider installing metallb.

## Configuring Routes

TODO

## Uninstalling HTNN

```shell
helm uninstall htnn-controller -n istio-system && helm uninstall htnn-gateway -n istio-system && kubectl delete ns istio-system
```
