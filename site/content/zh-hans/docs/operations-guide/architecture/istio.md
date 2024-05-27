---
title: Istio
---

## 介绍

Istio 是 HTNN 控制面的主要组件。除此之外，HTNN 还实现了自己的 controller 来调和 HTTPFilterPolicy 等 CRD。HTNN 通过 patch 将自己的 controller 和 Istio 结合在一起。Patch 里面包含下述功能：

* 监听 HTNN 自己的 CRD。
* 在 `PushContext` 里面，检测是否有需要调用 HTNN controller 来调和当前状态。
* HTNN controller 生成 EnvoyFilter 等 Istio 资源，写入到 Istio 的 `ConfigStore` 当中。Istio 推送给 Envoy 的配置将包含这部分生成的资源。

另外 HTNN 还 backport 了一些 Istio 的 bugfix 到当前所用的 Istio 版本中。

HTNN 的 Istio 百分之百兼容 Istio 官方版本。所有 Istio 的功能都能在 HTNN 的 Istio 发行版上使用。比起原版 Istio，HTNN 具有如下改动：

* 使用了自己的控制面和数据面镜像。
* 添加了一些环境变量，见下表。
* 增加了自己的 CRD，以及围绕这些 CRD 的 RBAC 和 webhook 配置。

## HTNN 相关的环境变量

| 名称                               | 类型    | 默认值            | 说明                                                                                                                                                                        |
|------------------------------------|---------|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| PILOT_ENABLE_HTNN                  | Boolean | false             | 如果启用，Pilot 将监听 HTNN 资源                                                                                                                                           |
| PILOT_ENABLE_HTNN_STATUS           | Boolean | false             | 如果设置为 true，我们将上报状态信息到 HTNN 资源                                                                                                                                |
| PILOT_SCOPE_GATEWAY_TO_NAMESPACE   | Boolean | false             | 此环境变量在 HTNN 中被设置为 true。我们假设 workload 的命名空间等于 gateway 的命名空间，以此减少管理 workload 命名空间的复杂性。                                                        |
| HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS    | Boolean | false             | 启用基于 ECDS 发布 LDS 插件的能力                                                                                                                                             |
| HTNN_ENVOY_GO_SO_PATH              | String  | /etc/libgolang.so | 数据面镜像中 Go 共享库的路径                                                                                                                                              |
| HTNN_ENABLE_NATIVE_PLUGIN          | Boolean | true              | 允许通过 HTNN 控制器配置 Native 插件                                                                                                                                    |
| HTNN_ENABLE_EMBEDDED_MODE           | Boolean | true              | 启用[嵌入模式](../../concept/embedded_mode)                                                                                                                               |
| HTNN_USE_WILDCARD_IPV6_IN_LDS_NAME | Boolean | false             | 在 LDS 名称中使用通配符 IPv6 地址作为默认前缀。如果你的网关默认监听 IPv6 地址，请开启此项。                                                                              | 
