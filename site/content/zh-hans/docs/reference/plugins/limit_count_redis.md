---
title: Limit Count Redis
---

## 说明

`limitCountRedis` 插件通过将统计数据存储在 Redis 上，实现了全局的固定窗口限流。用户可以使用该插件控制给定时间段内不同维度下的客户端访问次数。

## 属性

|       |         |
| ----- | ------- |
| Type  | Traffic |
| Order | Traffic |

## 配置

| 名称                    | 类型                                | 必选 | 校验规则                   | 说明                                                                                |
| ----------------------- | ----------------------------------- | ---- | -------------------------- | ----------------------------------------------------------------------------------- |
| address                 | string                              | 是   |                            | Redis 地址                                                                          |
| cluster                 | Cluster                             | 是   |                            | Redis cluster 配置。`address` 和`cluster` 只能配置一个。                            |
| rules                   | Rule                                | 是   | min_items: 1, max_items: 8 | 规则                                                                                |
| failureModeDeny         | bool                                | 否   |                            | 默认情况下，如果访问 Redis 失败，会放行请求。该值为 true 时，会拒绝请求。           |
| enableLimitQuotaHeaders | bool                                | 否   |                            | 是否设置限流额度相关的响应头                                                        |
| username                | string                              | 否   |                            | 用于访问 Redis 的用户名                                                             |
| password                | string                              | 否   |                            | 用于访问 Redis 的密码                                                               |
| tls                     | bool                                | 否   |                            | 是否通过 TLS 访问 Redis                                                             |
| tlsSkipVerify           | bool                                | 否   |                            | 通过 TLS 访问 Redis 时是否跳过验证                                                  |
| statusOnError           | [StatusCode](../../type#statuscode) | 否   |                            | 当无法访问 Redis 且 `failureModeDeny` 为 true 时，拒绝请求使用的状态码。默认为 500. |
| rateLimitedStatus       | [StatusCode](../../type#statuscode) | 否   |                            | 因限流产生的拒绝响应的状态码。默认为 429. 该配置仅在不小于 400 时生效。             |

每个规则的统计是独立的。当任一规则的额度用完后，就会触发限流操作。因限流产生的拒绝的响应中会包含 header `x-envoy-ratelimited: true`。如果配置了 `enableLimitQuotaHeaders` 为 `true`，所有响应中都会包括下面三个头：

* `x-ratelimit-limit`：表示当前应用的限流规则。格式为“当前剩余额度最少的规则, (规则额度;w=时间窗口){1个或多个规则}”，例如 `2, 2;w=60`。
* `x-ratelimit-remaining`：表示当前剩余额度最少的规则的剩余额度，最小值为 `0`。
* `x-ratelimit-reset`：表示当前剩余额度最少的规则什么时候重置，单位为秒，例如 `59`。注意由于网络延迟等原因，该值并非绝对精准。

### Cluster

| 名称      | 类型     | 必选 | 校验规则     | 说明       |
| --------- | -------- | ---- | ------------ | ---------- |
| addresses | string[] | 是   | min_items: 1 | Redis 地址 |

### Rule

| 名称       | 类型                            | 必选 | 校验规则 | 说明                                                                          |
| ---------- | ------------------------------- | ---- | -------- | ----------------------------------------------------------------------------- |
| timeWindow | [Duration](../../type#duration) | 是   | >= 1s    | 时间窗口                                                                      |
| count      | uint32                          | 是   | >= 1     | 次数                                                                          |
| key        | string                          | 否   |          | 用来作为限流的 key。默认是客户端 IP。这里可以使用 [CEL 表达式](../../expr) 。 |

请求数默认按客户端 IP 计数。你也可以通过配置 `key` 来使用别的字段。`key` 里面的配置会被作为 CEL 表达式解析。比如 `key: request.header("x-key")` 表示使用请求头 `x-key` 作为限流的维度。如果 `key` 对应值为空，则回退到使用客户端 IP 计数。

## 用法

首先，让我们先启动一个 Redis，监听 6379 端口。

假设我们提供了如下配置到 `http://localhost:10000/`：

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
---
apiVersion: htnn.mosn.io/v1
kind: HTTPFilterPolicy
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
        address: "0.0.0.0:6379"
        enableLimitQuotaHeaders: true
        failureModeDeny: true
        rules:
        - count: 2
            timeWindow: "60s"
```

对 `/echo` 路径发起请求，前两个请求将成功，随后的请求将失败：

```
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

```
$ curl http://localhost:10000/echo -i
HTTP/1.1 200 OK
x-ratelimit-limit: 2, 2;w=60
x-ratelimit-remaining: 1
x-ratelimit-reset: 60
date: Tue, 20 Feb 2024 03:54:16 GMT
```
