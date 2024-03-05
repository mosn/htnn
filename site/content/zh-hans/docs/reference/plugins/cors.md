---
title: CORS
---

## 说明

`cors` 插件通过利用 Envoy 的 `cors` 过滤器处理跨域资源共享的请求。

## 属性

|       |          |
|-------|----------|
| Type  | Security |
| Order | Outer    |

## 配置

请参阅相应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.28.0/configuration/http/http_filters/cors_filter)。

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
    namespace: default
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend
      port: 8080
```

通过应用下面的配置，发送到 `http://localhost:10000/` 的处理跨域资源共享的请求将会被处理。如果 OPTIONS 请求的请求头的 `Origin` 匹配到正则表达式 `.*\.default\.local`，那么对应的响应中会有配置的 `Access-Control-Allow-*` 响应头。

```yaml
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    cors:
      config:
        allowOriginStringMatch:
        - safeRegex:
            regex: ".*\.default\.local"
        allowMethods: POST
```

让我们试一下：

```
$ curl -X OPTIONS http://localhost:10000/ -H "Origin: https://x.efault.local" -H "Access-Control-Request-Method: GET" -i
HTTP/1.1 200 OK
server: istio-envoy
...

$ curl -X OPTIONS http://localhost:10000/ -H "Origin: https://x.default.local" -H "Access-Control-Request-Method: GET" -i
HTTP/1.1 200 OK
access-control-allow-origin: https://x.default.local
access-control-allow-methods: POST
server: istio-envoy
...
```
