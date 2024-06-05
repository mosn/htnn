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

安装命令：

```shell
$ helm install $package_name htnn/$package_name --namespace istio-system --create-namespace --wait
```

其中 `$package_name` 可以是：

* `htnn-controller`：控制面组件
* `htnn-gateway`：数据面组件

### 配置

我们可以使用 Helm 的 [Value files](https://helm.sh/docs/chart_template_guide/values_files/) 来配置 Helm Chart 的默认值。例如：

```shell
helm install htnn-controller htnn/htnn-controller ... --set istiod.pilot.env.HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS=true
```

控制面相关的配置项以 `istiod` 开头，具体配置项请参考 [控制面配置](https://github.com/istio/istio/blob/1.21.2/manifests/charts/istio-control/istio-discovery/values.yaml)。

```shell
helm install htnn-gateway htnn/htnn-gateway ... --set gateway.podAnnotations.test=ok
```

数据面相关的配置项以 `gateway` 开头，具体配置项请参考 [数据面配置](https://github.com/istio/istio/blob/1.21.2/manifests/charts/gateway/values.yaml)。
