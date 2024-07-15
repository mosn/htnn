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
...
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
...
NAME: htnn-gateway
LAST DEPLOYED: Wed May 29 19:59:22 2024
NAMESPACE: istio-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

Here we have not used the `--wait` parameter, instead, we used the `kubectl wait` command to wait for the `istio-ingressgateway` deployment to complete. Because `kind` does not support LoadBalancer type of Service by default, the ExternalIP for Service `istio-ingressgateway` will remain in `Pending` status. This does not affect our hands-on experience. If you're interested in this, refer to the [kind official documentation](https://kind.sigs.k8s.io/docs/user/loadbalancer/) and consider installing metallb.

## Configuring Routes

After installing HTNN, let's experience its features.

In this guide, we'll showcase HTNN's rate-limiting capabilities based on Redis. To do this, let's first deploy a Redis service:

```shell
kubectl apply -f https://raw.githubusercontent.com/mosn/htnn/main/examples/quick_start/redis.yaml && \
    kubectl wait --timeout=5m -n istio-system deployment/redis --for=condition=Available
```

We also need to deploy a backend service, here we use a simple echo service:

```shell
kubectl apply -f https://raw.githubusercontent.com/mosn/htnn/main/examples/quick_start/backend.yaml && \
    kubectl wait --timeout=5m -n istio-system deployment/backend --for=condition=Available
```

Next, we need to configure routing rules to forward requests to the backend service:

```shell
kubectl apply -f https://raw.githubusercontent.com/mosn/htnn/main/examples/quick_start/route.yaml
```

Everything is ready, we can now apply HTNN's policies:

```shell
kubectl apply -f - <<EOF
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
  namespace: istio-system
spec:
  targetRef:
    group: networking.istio.io
    kind: VirtualService
    name: vs
  filters:
    limitCountRedis:
      config:
        address: "redis:6379"
        rules:
        - count: 1
          timeWindow: "60s"
          key: |
            request.header("User") != "" ? request.header("User") : source.ip()
EOF
```

This policy adds the `limitCountRedis` plugin to the route. This plugin is implemented in Go and supports using Redis as a rate-limiting store. In this example, we limit each user to a maximum of 1 visit per minute. The user's identifier is the value in the `User` request header, if omitted, it is limited by the client's IP. You can view more information about the `limitCountRedis` plugin in the [plugin documentation](../reference/plugins/limit_count_redis.md).

## Verify Configuration

First, check the status to see if the policy has been accepted:

```shell
$ kubectl -n istio-system get filterpolicies.htnn.mosn.io policy -o yaml
...
status:
  conditions:
    ...
    message: The policy has been accepted
    observedGeneration: 1
    reason: Accepted
```

Let's execute port-forward in a terminal to allow the local client to access the service inside k8s:

```shell
kubectl port-forward -n istio-system pod/"$(kubectl -n istio-system get pods | grep '^istio-ingressgateway' |  cut -d' ' -f 1)" 18000:18000
```

In another terminal, we can access HTNN through port 18000:

```shell
$ curl --resolve default.local:18000:127.0.0.1 'http://default.local:18000/' -i
HTTP/1.1 200 OK
...
$ curl --resolve default.local:18000:127.0.0.1 'http://default.local:18000/' -i
HTTP/1.1 429 Too Many Requests
...
# Switch to another user
$ curl --resolve default.local:18000:127.0.0.1 'http://default.local:18000/' -i -H "User: someone else"
HTTP/1.1 200 OK
...
$ curl --resolve default.local:18000:127.0.0.1 'http://default.local:18000/' -i -H "User: someone else"
HTTP/1.1 429 Too Many Requests
...
```

As you can see, the rate-limiting logic is already effective.

## Configure Using Gateway API

HTNN also supports configuration using the Gateway API. Currently, Gateway API CRDs need to be installed separately:

```shell
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
```

Let's use the Gateway API to create routing rules to forward requests to the backend service:

```shell
kubectl apply -f https://raw.githubusercontent.com/mosn/htnn/main/examples/quick_start/route_gateway_api.yaml
```

Next, apply HTNN's configuration to the route created by Gateway API:

```shell
kubectl apply -f - <<EOF
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy-gateway-api
  namespace: istio-system
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: route
  filters:
    limitCountRedis:
      config:
        address: "redis:6379"
        rules:
        - count: 1
          timeWindow: "60s"
          key: |
            request.header("User") != "" ? request.header("User") : source.ip()
EOF
```

This policy is identical to the one above, only the targetRef now points to a Gateway API resource.

Continue to check the status to see if the policy has been accepted:

```shell
$ kubectl -n istio-system get filterpolicies.htnn.mosn.io policy-gateway-api -o yaml
...
status:
  conditions:
    ...
    message: The policy has been accepted
    observedGeneration: 1
    reason: Accepted
```

Let's execute port-forward in a terminal to allow the local client to access the service inside k8s:

```shell
kubectl port-forward -n istio-system pod/"$(kubectl -n istio-system get pods | grep '^default-istio' |  cut -d' ' -f 1)" 18001:18001
```

In another terminal, we can access HTNN through port 18001:

```shell
$ curl --resolve default.local:18001:127.0.0.1 'http://default.local:18001/' -i
HTTP/1.1 200 OK
...
$ curl --resolve default.local:18001:127.0.0.1 'http://default.local:18001/' -i
HTTP/1.1 429 Too Many Requests
...
```

## Uninstalling HTNN

```shell
helm uninstall htnn-controller -n istio-system && \
helm uninstall htnn-gateway -n istio-system && \
kubectl delete ns istio-system
```
