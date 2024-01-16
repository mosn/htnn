---
title: 消费者
---

在使用 API 网关时，我们经常需要把同样的认证逻辑应用到多个路由。因此 HTNN 提供了 `Consumer` 的概念，允许用户将通用的认证配置和与之相关的延伸操作放置到专门的 CRD 当中，更好地管理 API 资产。

举个例子，假设我们现有一个消费者 `Leo`，和两个路由。由于历史遗留原因，这两个路由上的认证参数，一个来自于 url，另一个来自于 header。如下所示：

```yaml
apiVersion: mosn.io/v1
kind: Consumer
metadata:
  name: leo
spec:
  auth:
    keyAuth:
      config:
        key: Leo
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: alpha
spec:
  parentRefs:
  - name: default
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend1
      port: 8080
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: beta
spec:
  parentRefs:
  - name: default
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend2
      port: 8081
---
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: alpha
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: alpha
  filters:
    keyAuth:
      keys:
        - name: ak
          source: query
---
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: beta
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: beta
  filters:
    keyAuth:
      keys:
        - name: Authorization
          source: header
```

如果没有消费者这一层抽象，那么每个路由都需要配置 `key: Leo`。假设有一天，比如说 `Leo` 只是个临时用户，我们需要回收权限。在有消费者的情况下，我们只需删除消费者 `Leo`，无需改动任何路由配置。

除此之外，我们还可以给消费者配置特定的插件。这些插件只有在通过认证之后才会执行。以下面的配置为例：

```yaml
apiVersion: mosn.io/v1
kind: Consumer
metadata:
  name: vip
spec:
  auth:
    keyAuth:
      config:
        key: vip
  filters:
    limitReq:
      config:
        average: 10
---
apiVersion: mosn.io/v1
kind: Consumer
metadata:
  name: member
spec:
  auth:
    keyAuth:
      config:
        key: member
  filters:
    limitReq:
      config:
        average: 1
---
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: beta
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: beta
  filters:
    keyAuth:
      keys:
        - name: Authorization
          source: header
```

如果认证结果是尊贵的 VIP 会员，那么 `average` 的配置会是 10。如果是普通的会员，那么对应的配置只是 1。

和 Kong / APISIX 里面的消费者不同的是，HTNN 的消费者是 `namespace` 级别的。来自不同 `namespace` 的消费者，只会应用到对应 `namespace` 里的路由配置（HTTPRoute、VirtualService 等等）里的路由。这种设计避免了不同业务间的消费者发生冲突。