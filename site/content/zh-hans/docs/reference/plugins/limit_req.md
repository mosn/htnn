---
title: Limit Req
---

## 说明

`limitReq` 插件限制了每秒对此代理的请求数量。其实现是基于[令牌桶算法](https://en.wikipedia.org/wiki/Token_bucket)。为易于理解，可以把（下文提到的） `average` 和 `period` 看做是桶的补充速率，而 `burst` 则代表桶的容量。

## 属性

|        |              |
|--------|--------------|
| Type   | Traffic      |
| Order  | Traffic      |
| Status | Experimental |

## 配置

| 名称    | 类型                            | 必选 | 校验规则 | 说明                                                                           |
|---------|---------------------------------|------|----------|--------------------------------------------------------------------------------|
| average | uint32                          | 是   | > 0      | 阈值，默认单位为每秒请求数计                                                   |
| period  | [Duration](../type.md#duration) | 否   |          | 速率的时间单位。限制速率定义为 `average / period`。默认为 1 秒，即每秒请求数。 |
| burst   | uint32                          | 否   |          | 允许超出速率的请求数。默认为 1。                                               |
| key     | string                          | 否   |          | 用来作为限流的 key。默认是客户端 IP。这里可以使用 [CEL 表达式](../expr.md) 。     |

当请求速率超过 `average / period`，且超出的请求数超过 `burst` 时，我们会计算降低速率至预期水平所需的延迟时间。如果所需延迟时间不大于最大延迟，则请求会被延迟。如果所需延迟大于最大延迟，则请求会以 `429` HTTP 状态码被丢弃。默认情况下，最大延迟是速率的一半（`1 / 2 * average / period`），如果 `average / period` 小于 1，则为 500 毫秒。

请求数默认按客户端 IP 计数。你也可以通过配置 `key` 来使用别的字段。`key` 里面的配置会被作为 CEL 表达式解析。比如 `key: request.header("x-key")` 表示使用请求头 `x-key` 作为限流的维度。如果 `key` 对应值为空，则回退到使用客户端 IP 计数。你也可以在表达式里提供默认值，比如 `key: 'request.header("x-key") != "" ? request.header("x-key") : request.header("x-forwarded-for")'` 表示先用请求头 `x-key` 作为限流的维度，找不到则改用 `x-forwarded-for`。

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
    limitReq:
      config:
        average: 1 # 限制请求到每秒 1 次
```

第一个请求将获得 `200` 状态码，随后的请求将以 `429` 被丢弃：

```shell
$ while true; do curl -I http://localhost:10000/ 2>/dev/null | head -1 ; done
HTTP/1.1 200 OK
HTTP/1.1 429 Too Many Requests
HTTP/1.1 429 Too Many Requests
```

如果客户端将其请求速率降低到每秒一个以下，所有请求都不会被丢弃：

```shell
$ while true; do curl -I http://localhost:10000/ 2>/dev/null | head -1 ; sleep 1; done
HTTP/1.1 200 OK
HTTP/1.1 200 OK
```
