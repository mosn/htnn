---
title: 快速上手
---

## 前提条件

* Kubernetes 1.26.0 或更高版本。推荐使用 [kind](https://kind.sigs.k8s.io/) 在本地快速搭建 Kubernetes 集群。
* Helm 3.6 或更高版本。安装 Helm 请参考 [Helm 安装指南](https://helm.sh/docs/intro/install/)。
* 配置 helm 仓库地址。执行以下命令添加仓库：

```shell
helm repo add htnn https://mosn.github.io/htnn
helm repo update
```

## 安装 HTNN

让我们把 HTNN 安装到 `istio-system` namespace 中。为了简单起见，HTNN 和其他用于 demo 的资源都会安装到该 namespace。

1. 安装控制面组件：

```shell
$ helm install htnn-controller htnn/htnn-controller \
    --set global.hub=m.daocloud.io/ghcr.io/mosn \
    --namespace istio-system --create-namespace --debug --wait
...
NAME: htnn-controller
LAST DEPLOYED: Wed May 29 18:42:18 2024
NAMESPACE: istio-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

2. 安装数据面组件：

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

这里我们没有使用 `--wait` 参数，而是使用 `kubectl wait` 命令等待 `istio-ingressgateway` 部署完成。因为 `kind` 默认不支持 LoadBalancer 类型的 Service，所以 Service `istio-ingressgateway` 的 ExternalIP 会一直处于 `Pending` 状态。这不影响我们的上手体验。如果你对此感兴趣，可以参考 [kind 官方文档](https://kind.sigs.k8s.io/docs/user/loadbalancer/) 以及安装 metallb。

## 配置路由

安装完 HTNN 后，让我们体验下它的功能。

在本指南中，我们将展示 HTNN 基于 Redis 的限流能力。为此，让我们先部署一个 Redis 服务：

```shell
kubectl apply -f https://raw.githubusercontent.com/mosn/htnn/main/examples/quick_start/redis.yaml && \
    kubectl wait --timeout=5m -n istio-system deployment/redis --for=condition=Available
```

我们还需要部署一个后端服务，这里我们使用一个简单的 echo 服务：

```shell
kubectl apply -f https://raw.githubusercontent.com/mosn/htnn/main/examples/quick_start/backend.yaml && \
    kubectl wait --timeout=5m -n istio-system deployment/backend --for=condition=Available
```

接下来，我们需要配置路由规则，将请求转发到后端服务：

```shell
kubectl apply -f https://raw.githubusercontent.com/mosn/htnn/main/examples/quick_start/route.yaml
```

一切准备就绪，我们就可以应用 HTNN 自己的策略了：

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

这个策略在路由上添加了 `limitCountRedis` 插件。该插件使用 Go 实现，支持 Redis 作为限流存储。在这个例子中，我们限制每个用户每分钟最多访问 1 次。用户的标识是 `User` 请求头中的值，如果没有就按客户端 IP 限流。你可以在 [插件文档](../reference/plugins/limit_count_redis.md) 查看更多关于 `limitCountRedis` 插件的信息。

## 验证配置

先通过 status 检查下策略是否被接受：

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

让我们在一个终端上执行 port-forward，让本地的客户端可以访问到 k8s 里面的服务：

```shell
kubectl port-forward -n istio-system pod/"$(kubectl -n istio-system get pods | grep '^istio-ingressgateway' |  cut -d' ' -f 1)" 18000:18000
```

在另一个终端上，我们可以通过 18000 端口访问到 HTNN：

```shell
$ curl --resolve default.local:18000:127.0.0.1 'http://default.local:18000/' -i
HTTP/1.1 200 OK
...
$ curl --resolve default.local:18000:127.0.0.1 'http://default.local:18000/' -i
HTTP/1.1 429 Too Many Requests
...
# 切换到另一个用户
$ curl --resolve default.local:18000:127.0.0.1 'http://default.local:18000/' -i -H "User: someone else"
HTTP/1.1 200 OK
...
$ curl --resolve default.local:18000:127.0.0.1 'http://default.local:18000/' -i -H "User: someone else"
HTTP/1.1 429 Too Many Requests
...
```

可以看到限流逻辑已经生效。

## 使用 Gateway API 配置

HTNN 也支持使用 Gateway API 配置。目前 Gateway API 的 CRD 需要单独安装：

```shell
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
```

让我们用 Gateway API 创建路由规则，将请求转发到后端服务：

```shell
kubectl apply -f https://raw.githubusercontent.com/mosn/htnn/main/examples/quick_start/route_gateway_api.yaml
```

接下来应用 HTNN 自己的配置到由 Gateway API 创建的路由上：

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

这个策略和前面的一模一样，只是在 targetRef 里指向了 Gateway API 资源。

继续通过 status 检查下策略是否被接受：

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

让我们在一个终端上执行 port-forward，让本地的客户端可以访问到 k8s 里面的服务：

```shell
kubectl port-forward -n istio-system pod/"$(kubectl -n istio-system get pods | grep '^default-istio' |  cut -d' ' -f 1)" 18001:18001
```

在另一个终端上，我们可以通过 18001 端口访问到 HTNN：

```shell
$ curl --resolve default.local:18001:127.0.0.1 'http://default.local:18001/' -i
HTTP/1.1 200 OK
...
$ curl --resolve default.local:18001:127.0.0.1 'http://default.local:18001/' -i
HTTP/1.1 429 Too Many Requests
...
```

## 卸载 HTNN

```shell
helm uninstall htnn-controller -n istio-system && \
helm uninstall htnn-gateway -n istio-system && \
kubectl delete ns istio-system
```
