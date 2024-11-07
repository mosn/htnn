---
title: Route Patch
---

## 说明

`routePatch` 插件允许用户直接给 VirtualService 或 HTTPRoute 对应的一个或多个 Envoy 路由打补丁。注意配置了 `routePatch` 插件的 FilterPolicy 的 TargetRef 只能是 VirtualService 或 HTTPRoute。

## 属性

|        |              |
|--------|--------------|
| Type   | General      |
| Order  | Inner        |
| Status | Experimental |

## 配置

请查阅对应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route_components.proto.html#envoy-v3-api-msg-config-route-v3-route)。

## 用法

假设我们有下面附加到 `localhost:10000` 的 HTTPRoute，并且有一个后端服务器监听端口 `8080`：

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: default
spec:
  parentRefs:
  - name: default
  rules:
  - matches:
    - path:
        type: Exact
        value: /reply
    backendRefs:
    - name: backend
      port: 8080
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend
      port: 8080
```

通过应用下面的配置，我们可以用自定义响应中断对 `http://localhost:10000/` 的请求：

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    routePatch:
      config:
        directResponse:
          status: 403
```

此时由 HTTPRoute `default` 生成的两条路由都会带上 direct_response 字段：

```yaml
routes:
    - directResponse:
        status: 403
      match:
        caseSensitive: true
        path: /reply
      name: e2e.default.0
      ...
    - directResponse:
        status: 403
      match:
        caseSensitive: true
        prefix: /
      name: e2e.default.1
      ...
```

我们也可以指定具体路由的名字，让修改只对具体路由生效。在下面配置中只有名字为 `last` 的路由才会被打上补丁：

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: vs
  namespace: istio-system
spec:
  gateways:
  - default
  hosts:
  - "default.local"
  http:
  - match:
    - uri:
        prefix: /echo
    route:
    - destination:
        host: backend
        port:
          number: 8080
  - match:
    - uri:
        prefix: /
    name: last
    route:
    - destination:
        host: backend
        port:
          number: 8080
---
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
    sectionName: last
  filters:
    routePatch:
      config:
        directResponse:
          status: 403
```
