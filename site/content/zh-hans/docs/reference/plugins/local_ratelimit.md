---
title: Local Ratelimit
---

## 说明

`localRatelimit` 插件通过利用 Envoy 的 `local_ratelimit` 过滤器，限制了每秒请求的数量。该插件在运行认证插件之前运行。

## 属性

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Outer   |

## 配置

请参阅相应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/http/http_filters/local_rate_limit_filter)。

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

通过应用下述配置，`http://localhost:10000/` 的请求速率被限制为每秒 1 个请求：

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
    localRatelimit:
      config:
        statPrefix: http_policy_local_rate_limiter
        tokenBucket:
          maxTokens: 1
          tokensPerFill: 1
          fillInterval: 1s
        filterEnabled:
          defaultValue:
            numerator: 100
            denominator: HUNDRED
        filterEnforced:
          defaultValue:
            numerator: 100
            denominator: HUNDRED
```

我们可以进行测试：

```
$ while true; do curl -I http://localhost:10000/ 2>/dev/null | head -1 ;done
HTTP/1.1 200 OK
HTTP/1.1 429 Too Many Requests
HTTP/1.1 429 Too Many Requests
```

```
$ while true; do curl -I http://localhost:10000/ 2>/dev/null | head -1 ; sleep 1; done
HTTP/1.1 200 OK
HTTP/1.1 200 OK
```
