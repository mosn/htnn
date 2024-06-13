---
title: 安装
---

## 通过 Helm 安装

### 前提条件

* Helm 3.6 或更高版本。安装 Helm 请参考 [Helm 安装指南](https://helm.sh/docs/intro/install/)。
* 配置 helm 仓库地址。执行以下命令添加仓库：

```shell
helm repo add htnn https://mosn.github.io/htnn
helm repo update
```

### 安装

控制面安装命令：

```shell
helm install htnn-controller htnn/htnn-controller \
    --set global.hub=m.daocloud.io/ghcr.io/mosn \
    --namespace istio-system --create-namespace --debug --wait
```

数据面安装命令：

```shell
helm install htnn-gateway htnn/htnn-gateway --namespace istio-system --create-namespace --debug --wait
```

请注意，`htnn-gateway` 的 pod spec 将在运行时自动填充，使用与 [Sidecar 注入](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection) 相同的机制。

这意味着两件事：

1. 部署 `htnn-gateway` 的命名空间不能有 `istio-injection=disabled` 标签。有关更多信息，请参见 [控制注入策略](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/#controlling-the-injection-policy)。
2. 必须在安装 `htnn-controller` 之后安装 `htnn-gateway`，以便可以注入 pod spec。

如果使用 `kind` 设置 k8s 环境，`helm install ... htnn/htnn-gateway --wait` 会失败，因为 `kind` 默认不支持 LoadBalancer service。可以使用 `kubectl wait --timeout=5m -n istio-system deployment/istio-ingressgateway --for=condition=Available` 来指示安装是否完成。

### 配置

我们可以使用 Helm 的 [Value files](https://helm.sh/docs/chart_template_guide/values_files/) 来配置 Helm Chart 的默认值。例如：

```shell
helm install htnn-controller htnn/htnn-controller ... --set istiod.pilot.env.HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS=true
```

htnn-controller 具体配置项请参考 [htnn-controller 配置](https://artifacthub.io/packages/helm/htnn/htnn-controller#values)。

```shell
helm install htnn-gateway htnn/htnn-gateway ... --set gateway.podAnnotations.test=ok
```

htnn-gateway 相关的配置项以 `gateway` 开头，具体配置项请参考 [数据面配置](https://github.com/istio/istio/blob/1.21.3/manifests/charts/gateway/values.yaml)。
