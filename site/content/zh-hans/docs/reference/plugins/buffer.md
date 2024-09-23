---
title: Buffer
---

## 说明

`buffer` 插件通过利用 Envoy 的 `buffer` 过滤器完全缓冲整个请求。它可以用于几个目的，比如：

* 保护应用程序免受高网络延迟的影响。
* 一些插件会缓冲整个请求。当请求体超过最大长度时，将触发 413 HTTP 状态码。我们可以设置 `maxRequestBytes` 大于 `per_connection_buffer_limit_bytes` 来增加单个路由的最大请求体大小。

## 属性

|       |         |
|-------|---------|
| Type  | General |
| Order | Outer   |

## 配置

| 名称            | 类型   | 必选 | 校验规则 | 说明                                                                                |
|-----------------|--------|------|----------|-------------------------------------------------------------------------------------|
| maxRequestBytes | uint32 | 是   |          | 缓冲的请求体最大大小，超过此大小将返回 413 响应。                                      |

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

通过应用下面的配置，发送到 `http://localhost:10000/` 的请求会被缓冲，直到请求体长度达到最大请求字节数 4：

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
    buffer:
      config:
        maxRequestBytes: 4
```

我们可以测试一下：

```shell
$ curl -d "hello" http://localhost:10000/ -i
HTTP/1.1 413 Payload Too Large
```

```shell
$ curl -d "hell" http://localhost:10000/ -i
HTTP/1.1 200 OK
```
