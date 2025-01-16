---
title: Limit Count Redis
---

## 说明

`limitCountRedis` 插件通过将统计数据存储在 Redis 上，实现了全局的固定窗口限流。用户可以使用该插件控制给定时间段内不同维度下的客户端访问次数。

## 属性

|        |         |
|--------|---------|
| Type   | Traffic |
| Order  | Traffic |
| Status | Stable  |

## 配置

| 名称                             | 类型                                  | 必选  | 校验规则                       | 说明                                                                                                                                                          |
|--------------------------------|-------------------------------------|-----|----------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| address                        | string                              | 否   |                            | Redis 地址。`address` 和`cluster` 只能配置一个。                                                                                                                       |
| cluster                        | Cluster                             | 否   |                            | Redis cluster 配置。`address` 和`cluster` 只能配置一个。                                                                                                               |
| prefix                         | string                              | 是   | min_len: 1, max_len: 128   | 该字段将用作 Redis key 的前缀。引入这个字段是为了在重新创建路由时不会重置计数器，因为新的限制统计将使用与前一个相同的 key。通常，用一个随机字符串作为它的值就够了。要在多条路由中共享计数器，我们可以使用相同的前缀。在这种情况下，请确保这些路由的 `limitCountRedis` 插件配置相同。 |
| rules                          | Rule                                | 是   | min_items: 1, max_items: 8 | 规则                                                                                                                                                          |
| failureModeDeny                | bool                                | 否   |                            | 默认情况下，如果访问 Redis 失败，会放行请求。该值为 true 时，会拒绝请求。                                                                                                                 |
| enableLimitQuotaHeaders        | bool                                | 否   |                            | 是否设置限流额度相关的响应头                                                                                                                                              |
| username                       | string                              | 否   |                            | 用于访问 Redis 的用户名                                                                                                                                             |
| password                       | string                              | 否   |                            | 用于访问 Redis 的密码                                                                                                                                              |
| tls                            | bool                                | 否   |                            | 是否通过 TLS 访问 Redis                                                                                                                                           |
| tlsSkipVerify                  | bool                                | 否   |                            | 通过 TLS 访问 Redis 时是否跳过验证                                                                                                                                     |
| statusOnError                  | [StatusCode](../type.md#statuscode) | 否   |                            | 当无法访问 Redis 且 `failureModeDeny` 为 true 时，拒绝请求使用的状态码。默认为 500.                                                                                                |
| rateLimitedStatus              | [StatusCode](../type.md#statuscode) | 否   |                            | 因限流产生的拒绝响应的状态码。默认为 429. 该配置仅在不小于 400 时生效。                                                                                                                   |
| disableXEnvoyRatelimitedHeader | bool                                | 否   |                            | 触发限流时是否关闭`x-envoy-ratelimited`的响应头                                                                                                                          |

每个规则的统计是独立的。当任一规则的额度用完后，就会触发限流操作。因限流产生的拒绝的响应中会包含 header `x-envoy-ratelimited: true`(可配置关闭)。如果配置了 `enableLimitQuotaHeaders` 为 `true` 且访问 Redis 成功，所有响应中都会包括下面三个头：

* `x-ratelimit-limit`：表示当前应用的限流规则。格式为“当前剩余额度最少的规则，(规则额度;w=时间窗口){1 个或多个规则}”，例如 `2, 2;w=60`。
* `x-ratelimit-remaining`：表示当前剩余额度最少的规则的剩余额度，最小值为 `0`。
* `x-ratelimit-reset`：表示当前剩余额度最少的规则什么时候重置，单位为秒，例如 `59`。注意由于网络延迟等原因，该值并非绝对精准。

### Cluster

| 名称      | 类型     | 必选 | 校验规则     | 说明       |
| --------- | -------- | ---- | ------------ | ---------- |
| addresses | string[] | 是   | min_items: 1 | Redis 地址 |

### Rule

| 名称       | 类型                            | 必选 | 校验规则 | 说明                                                                          |
| ---------- | ------------------------------- | ---- | -------- | ----------------------------------------------------------------------------- |
| timeWindow | [Duration](../type.md#duration) | 是   | >= 1s    | 时间窗口                                                                      |
| count      | uint32                          | 是   | >= 1     | 次数                                                                          |
| key        | string                          | 否   |          | 用来作为限流的 key。默认是客户端 IP。这里可以使用 [CEL 表达式](../expr.md) 。 |

请求数默认按客户端 IP 计数。你也可以通过配置 `key` 来使用别的字段。`key` 里面的配置会被作为 CEL 表达式解析。比如 `key: request.header("x-key")` 表示使用请求头 `x-key` 作为限流的维度。如果 `key` 对应值为空，则回退到使用客户端 IP 计数。

## 用法

首先，让我们假设现在有一个 Redis 服务 `redis.service` 正在监听 6379 端口。

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
    limitCountRedis:
      config:
        prefix: "47d26c6c"
        address: "redis.service:6379"
        enableLimitQuotaHeaders: true
        failureModeDeny: true
        rules:
        - count: 2
          timeWindow: "60s"
```

对 `/echo` 路径发起请求，前两个请求将成功，随后的请求将失败：

```shell
$ curl http://localhost:10000/echo -i
HTTP/1.1 200 OK
x-ratelimit-limit: 2, 2;w=60
x-ratelimit-remaining: 1
x-ratelimit-reset: 60
date: Tue, 20 Feb 2024 03:53:01 GMT
...
$ curl http://localhost:10000/echo -i
HTTP/1.1 200 OK
x-ratelimit-limit: 2, 2;w=60
x-ratelimit-remaining: 0
x-ratelimit-reset: 59
date: Tue, 20 Feb 2024 03:53:02 GMT
...
$ curl http://localhost:10000/echo -i
HTTP/1.1 429 Too Many Requests
x-envoy-ratelimited: true
x-ratelimit-limit: 2, 2;w=60
x-ratelimit-remaining: 0
x-ratelimit-reset: 58
date: Tue, 20 Feb 2024 03:53:03 GMT
...
```

过一分钟后，新的请求将成功：

```shell
$ curl http://localhost:10000/echo -i
HTTP/1.1 200 OK
x-ratelimit-limit: 2, 2;w=60
x-ratelimit-remaining: 1
x-ratelimit-reset: 60
date: Tue, 20 Feb 2024 03:54:16 GMT
```
