---
title: Embedded Mode
---

在有些场景下，将路由配置和策略配置分开下发会遇到一些挑战：

* 路由配置和策略配置有可能存在不一致的情况，比如 istio 已经 watch 到路由配置，但策略配置可能因为 API server 的限流导致无法写入。约束使用方以特定的顺序来使用路由和策略会比较复杂。
* 对于策略完全取决于路由配置的业务场景，业务方更希望能够像传统的网关那样，直接用一个 CRD 同时表达路由和插件配置，减少理解和维护成本。

为了解决上述问题，HTNN 提供了 Embedded Mode。

假如我们有这样一个 FilterPolicy：

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
  namespace: default
spec:
  filters:
    animal:
      config:
        kind: cat
  subPolicies:
  - sectionName: route1
    filters:
      animal:
        config:
          kind: goldfish
  - sectionName: route2
    filters:
      animal:
        config:
          kind: catfish
```

由于该资源只用于 annotation，应用目标是固定的，所以不需要额外指定 targetRef。
在使用时，用户将上述资源序列化成 JSON，放到 `htnn.mosn.io/filterpolicy` 这个 annotation 当中。如下所示：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: vs
  namespace: default
  annotations:
    htnn.mosn.io/filterpolicy: |
      {"apiVersion":"htnn.mosn.io/v1","kind":"FilterPolicy","metadata":{"name":"policy","namespace":"default"},"spec":{"filters":{"animal":{"config":{"kind":"cat"}}},"subPolicies":[{"sectionName":"route1","filters":{"animal":{"config":{"kind":"goldfish"}}}},{"sectionName":"route2","filters":{"animal":{"config":{"kind":"catfish"}}}]}}
spec:
  gateways:
  - default
  hosts:
  - default.local
  http:
  - match:
    - uri:
        prefix: /
    name: route
    route:
    - destination:
        host: httpbin
        port:
          number: 8000
```

控制面在收到 VirtualService 时，会查看它的 annotation `htnn.mosn.io/filterpolicy` 里是否有 FilterPolicy。如果有，则相当于同时收到 FilterPolicy 和它对应的 VirtualService。所以下发路由和策略时，只需要下发 VirtualService 即可。和 Ingress 的 annotation 不一样的是，这里面的 FilterPolicy 仍然会参与策略合并，所以用户还是可以指定一个更高级别的 FilterPolicy（比如作用于整个 Gateway），来添加额外的插件。

注意 Embedded Mode 目前只支持将 FilterPolicy 嵌入到 VirtualSrvice。嵌入 FilterPolicy 到 Istio Gateway 也是支持的，不过需要在控制面开启 `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS`。
