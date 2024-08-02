---
title: Ext Auth
---

## 说明

`extAuth` 插件向授权服务发送鉴权请求，以检查客户端请求是否得到授权。

## 属性

|       |       |
|-------|-------|
| Type  | Authz |
| Order | Authz |

## 配置

| 名称          | 类型          | 必选 | 校验规则 | 说明 |
|--------------|---------------|------|----------|------|
| httpService | HttpService   | 是   |          |      |
| failureModeAllow | bool | 否 | | 默认为 false。当设置为 true 时，即使与授权服务的通信失败，或者授权服务返回了 HTTP 5xx 错误，过滤器仍会接受客户端请求 |
| failureModeAllowHeaderAdd | bool | 否 | | 默认为 false。当 `failureModeAllow` 和 `failureModeAllowHeaderAdd` 都设置为 true 时，若与授权服务的通信失败，或授权服务返回了 HTTP 5xx 错误，那么请求头中将会添加 `x-envoy-auth-failure-mode-allowed: true` |

### HttpService

| 名称                  | 类型                                       | 必选 | 校验规则           | 说明                                                                                                                                                  |
|---------------------|--------------------------------------------|------|--------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------|
| url                   | string                                     | 是   | must be valid URI    | 外部服务的 uri，如 `http://ext_auth/prefix`。uri 的路径将作为鉴权请求路径的前缀。                                                                    |
| timeout               | [Duration](../type.md#duration)             | 否   | > 0s                 | 超时时长。例如，`10s` 表示超时时间为 10 秒。默认值为 0.2s。                                                                                             |
| authorizationRequest  | AuthorizationRequest                        | 否   |                      |                                                                                                                                                        |
| authorizationResponse | AuthorizationResponse                       | 否   |                      |                                                                                                                                                        |
| statusOnError         | [StatusCode](../type.md#statuscode)         | 否   |                      | 当鉴权服务器返回错误或无法访问时，设置返回给客户端的 HTTP 状态码。默认状态码是 `401`。                                                                   |
| withRequestBody       | bool                                       | 否   |                      | 缓冲客户端请求体，并将其发送至鉴权请求中。                                                                                                          |

### AuthorizationRequest

| 名称        | 类型                                             | 必选 | 校验规则           | 说明                                                                                                                                                        |
|------------|--------------------------------------------------|------|--------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| headersToAdd | [HeaderValue[]](../type.md#headervalue)           | 否   | min_items: 1       | 设置将包含在鉴权服务请求中的请求头列表。请注意，同名的客户端请求头将被覆盖。                                                                           |

### AuthorizationResponse

| 名称                   | 类型                                                   | 必选 | 校验规则           | 说明                                                                                                                                                                        |
|----------------------|-------------------------------------------------------|------|--------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| allowedUpstreamHeaders | [StringMatcher[]](../type.md#stringmatcher) | 否   | min_items: 1       | 当设置后，具有相应匹配项的鉴权请求的响应头将添加到原始的客户端请求头中。请注意，同名的请求头将被覆盖。                                                                                    |
| allowedClientHeaders   | [StringMatcher[]](../type.md#stringmatcher) | 否   | min_items: 1       | 当设置后，在请求被拒绝时，具有相应匹配项的鉴权请求的响应头将添加到客户端的响应头中。                                                                                                 |

## 用法

### 发送鉴权请求

每次客户端请求执行此插件都将触发鉴权请求。鉴权请求将包括：

* 原始的客户端请求的方法
* 原始的客户端请求的 `Host`
* 添加配置的前缀后的原始的客户端请求路径
* 原始的客户端请求头 `Authorization`

假设我们有下面附加到 `localhost:10000` 的 HTTPRoute，并且有一个后端服务器监听端口 `8080`：

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: default
spec:
  parentRefs:
  - name: default
  hostnames:
  - localhost
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend
      port: 8080
```

让我们应用下面的配置：

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
    extAuth:
      config:
        httpService:
          url: "http://127.0.0.1:10001/ext_auth"
```

如果我们使用名为 `users` 的路径进行 GET 请求：

```shell
curl -i http://localhost:10000/users -X GET -H "foo: bar" -H "Authorization: xxx"
```

监听 `10001` 的服务将接收到类似如下的鉴权请求：

```
GET /ext_auth/users HTTP/1.1
Host: localhost:10000
User-Agent: Go-http-client/1.1
Authorization: xxx
Accept-Encoding: gzip
```

可以了解到该请求具有和客户端请求相同的方法，并将前缀 `ext_auth` 添加到请求路径上。

如果客户端的请求带有请求体：

```shell
curl -i http://localhost:10000/users -d 'test'
```

鉴权请求将为：

```
POST /ext_auth/users HTTP/1.1
Host: localhost:10000
User-Agent: Go-http-client/1.1
Content-Length: 0
Accept-Encoding: gzip
```

如果配置了 `headersToAdd`，额外的头将被设置到鉴权请求中。

### 鉴权服务器响应

当服务器以 HTTP 状态 200 响应时，客户端请求将通过鉴权。如果配置了 `allowedUpstreamHeaders`，具有相应匹配项的响应头将作为请求头添加到原始的客户端请求中。

当服务器无法访问或状态码为 5xx 时，将以 `statusOnError` 配置的状态码拒绝客户端请求。

当服务器返回其他 HTTP 状态码时，将以返回的状态码拒绝客户端请求。如果配置了 `allowedClientHeaders`，具有相应匹配项的响应头将添加到客户端的响应中。
