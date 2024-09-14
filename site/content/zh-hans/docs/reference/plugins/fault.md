---
title: Fault
---

## 说明

`fault` 插件支持通过利用 Envoy 的 `fault` 过滤器注入响应和延迟。

## 属性

|       |         |
|-------|---------|
| Type  | General |
| Order | Outer   |

## 配置

请参阅相应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/http/http_filters/fault_filter)。

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
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend
      port: 8080
```

通过应用下面的配置，发送到 `http://localhost:10000/` 的请求将 100% 收到 401 响应：

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
    fault:
      config:
        abort:
          http_status: 401
          percentage:
            numerator: 100
```

让我们试一下：

```shell
$ curl http://localhost:10000/ -i 2>/dev/null | head -1
HTTP/1.1 401 Unauthorized
```
