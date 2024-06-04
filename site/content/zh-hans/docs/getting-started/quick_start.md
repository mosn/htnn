---
title: 快速上手
---

## 前提条件

* Kubernetes 1.26.0 或更高版本。推荐使用 [kind](https://kind.sigs.k8s.io/) 在本地快速搭建 Kubernetes 集群。
* Helm 3.6 或更高版本。安装 Helm 请参考 [Helm 安装指南](https://helm.sh/docs/intro/install/)。
* 配置 helm 仓库地址。执行以下命令添加仓库：

```shell
helm repo add mosn xxxx # TODO: setup such a repo
helm repo update
```

## 安装 HTNN

1. 安装控制面组件：

```shell
$ helm install htnn-controller mosn/htnn-controller --namespace istio-system --create-namespace --wait

NAME: htnn-controller
LAST DEPLOYED: Wed May 29 18:42:18 2024
NAMESPACE: istio-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

2. 安装数据面组件：

```shell
$ helm install htnn-gateway mosn/htnn-gateway --namespace istio-system --create-namespace && \
    kubectl wait --timeout=5m -n istio-system deployment/istio-ingressgateway --for=condition=Available

NAME: htnn-gateway
LAST DEPLOYED: Wed May 29 19:59:22 2024
NAMESPACE: istio-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

这里我们没有使用 `--wait` 参数，而是使用 `kubectl wait` 命令等待 `istio-ingressgateway` 部署完成。因为 kind 默认不支持 LoadBalancer 类型的 Service，所以 Service `istio-ingressgateway` 的 ExternalIP 会一直处于 `Pending` 状态。这不影响我们的上手体验。如果你对此感兴趣，可以参考 [kind 官方文档](https://kind.sigs.k8s.io/docs/user/loadbalancer/) 以及安装 metallb。

## 配置路由

TODO

## 卸载 HTNN

```shell
helm uninstall htnn-controller -n istio-system && helm uninstall htnn-gateway -n istio-system && kubectl delete ns istio-system
```
