---
title: FilterPolicy
---

绝大部分网关和 service mesh 上的业务需求，都是围绕着网络协议做一些事情，如认证鉴权、限流限速、请求改写等等。HTNN 把这部分的需求都抽象出来，使用 FilterPolicy 来表达具体的配置规则。

和一些同类产品不同，HTNN 并没有为不同的业务分类使用不同的 CRD，而是统一使用 FilterPolicy 一个 CRD 来解决所有的策略层面上的业务需求。这是因为我们觉得多 CRD 的成本太大了。我们甚至引入 `0 CRD` 的 [embedded mode](./embedded_mode.md)，来减低接入和维护成本。

## FilterPolicy 的结构说明

一个典型的 FilterPolicy 结构如下：

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  creationTimestamp: "2024-05-13T07:15:09Z"
  generation: 1
  name: policy
  namespace: istio-system
  resourceVersion: "158934"
  uid: 5b368582-0de3-4db0-b447-6c858b5a1305
spec:
  targetRef:
    group: networking.istio.io
    kind: VirtualService
    name: vs
    sectionName: to-httpbin
  filters:
    animal:
      config:
        pet: goldfish
    plant:
      config:
        vegetable: carrot
status:
  conditions:
  - lastTransitionTime: "2024-05-13T07:15:10Z"
    message: The policy targets non-existent resource
    observedGeneration: 1
    reason: TargetNotFound
    status: "False"
    type: Accepted
```

这个 FilterPolicy 里有一个 `targetRef`。`targetRef` 可以决定 FilterPolicy 针对哪种资源生效。目前我们支持的资源如下：

| group                     | kind           | 备注                                                           |
|---------------------------|----------------|----------------------------------------------------------------|
| networking.istio.io       | VirtualService |                                                                |
| networking.istio.io       | Gateway        | 需要控制面启用 `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS`。详情见下文。 |
| gateway.networking.k8s.io | HTTPRoute      |                                                                |
| gateway.networking.k8s.io | Gateway        | 需要控制面启用 `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS`。详情见下文。 |

`sectionName` 是可选的，仅在 `kind` 为 VirtualService 或 Gateway 时才生效。

* 当它作用于 VirtualService 时，可用于指定针对 VirtualService 下面的哪条路由生效。此时，`sectionName` 需要和 VirtualService 下面的某个路由的 `name` 字段匹配。注意如果同一个域名的多个 VirtualService 都设置了同名的路由，那么 istio 最终也会给该域名生成多条同名的路由，导致 FilterPolicy 实际上会命中其他 VirtualService 上的同名路由。所以对于同一域名的不同 VirtualService，需要避免出现同名的路由。
* 当它作用于 Gateway 时，可用于指定针对 Gateway 下面的哪个 Server 或者 Listener 生效。此时，`sectionName` 需要和 istio Gateway 下面的某个 Server 的 `name` 字段抑或 k8s Gateway 下面的某个 Listener 的 `name` 字段匹配。注意因为目前 Gateway 级策略的粒度最细到端口级别，所以实际上针对匹配到的 Server 或 Listener 所在的端口生效。

使用 `sectionName` 的具体示例见下文。

目前 FilterPolicy 只能作用于同 namespace 的路由资源，而且目标资源所在的 Gateway 需要和该资源位于同一个 namespace。

这个 FilterPolicy 还有一个 `filters`。`filters` 里面可以配置多个插件，如示例中的 `animal` 和 `plant`。每个插件的执行顺序，由注册插件时[指定的顺序](../developer-guide/plugin_development.md#插件顺序)决定。每个插件的具体配置，配置在该插件名下面的 `config` 字段里面。

和其他 k8s 资源一样，HTNN 控制面也会修改 FilterPolicy 的 `status` 字段，来报告这个 FilterPolicy 的状态。目前 `status` 字段下的 `reason` 为以下值之一：

| 名称           | 说明                                         |
|----------------|----------------------------------------------|
| TargetNotFound | 策略指定的资源不存在或不合法                 |
| Invalid        | 策略不合法                                   |
| Accepted       | 策略可以被调和（但不表示策略已在数据面生效） |

如果策略无法被调和，具体的错误信息会在 `message` 字段。

注意：重启或升级 HTNN 控制面不会主动重新检验不合法（`reason` 为 `Invalid`）的策略。如果你想触发重新检验（包括把曾经合法的策略变更成不合法的），需要手动重新创建策略。

## 在不同场景里使用 FilterPolicy 配置策略

对于按 API 维度配置的网关，我们可以给每个 API 定义一个 VirtualService，然后定一个 FilterPolicy 指向它：

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: VirtualService
  metadata:
    name: vs
    namespace: default
  spec:
    gateways:
    - default
    hosts:
    - default.local
    http:
    - match:
      - uri:
          exact: /v1/api
      route:
      - destination:
          host: httpbin
          port:
            number: 8000
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: VirtualService
      name: vs
    filters:
      animal:
        config:
          pet: goldfish
```

同一个域名下，不同 API 会使用不同的 VirtualService，也会有不同的 FilterPolicy。

对于按域名维度配置的网关，我们可以给每个域名定义一个 VirtualService，然后定一个 FilterPolicy 指向它：

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: VirtualService
  metadata:
    name: vs
    namespace: default
  spec:
    gateways:
    - default
    hosts:
    - default.local
    http:
    - match:
      - uri:
          exact: /httpbin
      name: to-httpbin
      route:
      - destination:
          host: httpbin
          port:
            number: 80
    - match:
      - uri:
          prefix: /
      route:
      - destination:
          host: default
          port:
            number: 8000
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: VirtualService
      name: vs
    filters:
      animal:
        config:
          pet: goldfish
```

如果想要给该域名下的特定路由下发特定的策略，那么需要在 FilterPolicy 中指定路由的名称。比如在上面示例中，有一个路由是

```yaml
  http:
    - match:
      - uri:
          exact: /httpbin
      name: to-httpbin
      route:
      - destination:
          host: httpbin
          port:
            number: 80
```

假设我们想要给这条 `to-httpbin` 路由下发策略，需要在 `sectionName` 中指定：

```yaml
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: to-httpbin-policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: VirtualService
      name: vs
      sectionName: to-httpbin
    filters:
      animal:
        config:
          pet: cat
```

对于 VirtualService 里的其他路由，“pet”对应的配置都是 goldfish。只有“to-httpbin”这条路由的配置是 cat。

我们也可以在 [LDS](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/lds.html) 级别上配置策略。简单来说，你可以理解成端口级别上的配置。要想配置 LDS 级别的策略，需要在启动控制面时设置环境变量 `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS` 为 true。

以下面的配置为例：

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: Gateway
  metadata:
    name: httpbin-gateway
    namespace: default
  spec:
    selector:
      istio: ingressgateway
    servers:
    - hosts:
      - '*'
      port:
        name: port-http
        number: 80
        protocol: HTTP
    - hosts:
      - '*'
      name: https
      port:
        name: port-https
        number: 443
        protocol: HTTPS
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: Gateway
      name: httpbin-gateway
    filters:
      animal:
        config:
          pet: goldfish
```

当我们下发一个指向 Gateway 的 FilterPolicy，由该 Gateway 生成的 LDS（这里指 80 和 443 两个端口）下的所有路由都会配置有插件 `animal`。

我们也可以通过 `sectionName` 来指定下发配置到 443 端口：

```yaml
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: Gateway
      name: httpbin-gateway
      sectionName: https
    filters:
      animal:
        config:
          pet: cat
```

注意由于目前我们只支持端口级别的 Gateway 配置，所以针对多个同端口的 section 配置的策略，其行为是未定义的。将来我们可能会支持更细粒度的配置。

以下面的配置为例：

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: Gateway
  metadata:
    name: httpbin-gateway
    namespace: default
  spec:
    selector:
      istio: ingressgateway
    servers:
    - hosts:
      - '*.test.com'
      name: test
      port:
        name: port-https
        number: 443
        protocol: HTTPS
    - hosts:
      - '*.example.com'
      name: example
      port:
        name: port-https
        number: 443
        protocol: HTTPS
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: Gateway
      name: httpbin-gateway
      sectionName: test
    filters:
      animal:
        config:
          pet: goldfish
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: Gateway
      name: httpbin-gateway
      sectionName: example
    filters:
      animal:
        config:
          pet: cat
```

最终结果里，443 端口上的 `pet` 既有可能是 goldfish，也有可能是 cat。目前我们不保证这种情况下的行为。

默认情况下，`HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS` 是禁用的，因为该功能会给每个 LDS 生成对应的 ECDS。

1. 在 LDS 数量众多的情况下，大量的 ECDS 可能会带来高开销。
2. 我们无法通过 ECDS 禁用一个 LDS 级别上的 HTTP filter。所以该 LDS 下的每条路由都会产生切换到 Go 上下文的开销。

如果以下情况适用于你，你可以启用它：

1. 如果你正在使用 HTNN 作为网关。
2. LDS 的数量有限。最好自己运行基准测试测试下性能，看看是否在预期内。
3. 你需要 LDS 级别的插件。

虽然我们在上述示例中使用的都是 Istio 的 CRD，但使用 Gateway API 也能完成同样的配置，只需更改 `targetRef` 的内容即可。

比如想要给 Gateway API 的 Gateway 资源的某个 Listener 下发策略，可以这样配置：

```yaml
- apiVersion: gateway.networking.k8s.io/v1
  kind: Gateway
  metadata:
    name: gw
    namespace: default
  spec:
    gatewayClassName: istio
    listeners:
    - name: http
      port: 80
      protocol: HTTP
    - name: http2
      port: 8080
      protocol: HTTP
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: gateway.networking.k8s.io
      kind: Gateway
      name: gw
      sectionName: http
    filters:
      limitReq:
        config:
          average: 1
```

生效范围重叠的不同的 FilterPolicy 配置的插件会合并，然后按注册插件时指定的顺序执行插件。
如果不同级别的 FilterPolicy 配置了同一个插件，那么范围更小的 FilterPolicy 上的配置会覆盖掉范围更大的配置，即 `SectionName` > `VirtualService/HTTPRoute` > `Gateway`。
如果同一级别的 FilterPolicy 配置了同一个插件，那么创建时间更早的 FilterPolicy 优先；如果时间都一样，则按 FilterPolicy 的 namespace 和 name 排序。

## 插件和 FilterPolicy 的对应关系

FilterPolicy 只是插件的载体。HTNN 的插件可以分成两类：

* 运行在数据面上的 Go 插件
* 运行在控制面上，生成对应 Envoy 配置的插件，我们称之为 Native 插件

Native 插件根据其作用位置不同，又可分成以下几类：

* HTTP Native 插件，作用在 HTTP filter 上
* Network Native 插件，作用在 Network filter 上
* Listener Native 插件，作用在 Listener 上

在每个插件的文档上，我们标注了它所属的类别。在“属性”这一节里，如果 `Order` 为

* `Listener`，则是 Listener Native 插件
* `Network`，则是 Network Native 插件
* `Outer` 或 `Inner`，则是 HTTP Native 插件
* 剩下的则是 Go 插件

一个 FilterPolicy 上能配置哪些插件取决于 `TargetRef` 里的目标资源类型，见下表：

| 插件类型             | 在 Gateway 上配置 | 在路由上配置 |
|----------------------|-------------------|--------------|
| Go 插件              | 支持              | 支持         |
| HTTP Native 插件     | 待支持            | 支持         |
| Network Native 插件  | 支持              | 不支持       |
| Listener Native 插件 | 支持              | 不支持       |

## 使用 subPolicies 减少 FilterPolicy 数量

对于按域名维度配置的网关，一个 VirtualService 内可能会有上百个路由。如果每个路由都需要有自己的配置，那么我们需要创建成百个 FilterPolicy。为了减少对 API server 的压力，我们支持使用同一个 FilterPolicy 匹配多个路由。

如下所示：

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: VirtualService
  metadata:
    name: vs
    namespace: default
  spec:
    gateways:
    - default
    hosts:
    - default.local
    http:
    - match:
      - uri:
          exact: /a
      name: route
      route:
      - destination:
          host: httpbin
          port:
            number: 8000
    - match:
      - uri:
          prefix: /
      name: route2
      route:
      - destination:
          host: httpbin
          port:
            number: 8000
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: VirtualService
      name: vs
    subPolicies:
    - sectionName: route
      filters:
        animal:
          config:
            pet: bird
    - sectionName: route2
      filters:
        animal:
          config:
            pet: dog
```

FilterPolicy 支持使用 `subPolicies` 字段同时给多个 `sectionName` 配置策略。`filters` 和 `subPolicies` 能同时使用，配置合并的规则和分开使用多个 FilterPolicy 一样。

注意目前 `subPolicies` 仅支持 VirtualService。
