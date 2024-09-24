---
title: Sentinel
---

## 说明

`sentinel` 插件利用 [sentinel-golang](https://github.com/alibaba/sentinel-golang) 来提供流量控制能力，当前提供以下三种配置规则：

- flow：流量控制
- hot spot：热点参数流控
- circuit breaker：熔断降级

## 属性

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Traffic |

## 配置

配置字段与 sentinel-golang v1.0.4 基本保持一致，并裁剪了部分无法被插件化的功能

详细配置说明请额外参考 [sentinel-golang 官方文档](https://sentinelguard.io/zh-cn/docs/golang/quick-start.html)

| 名称             | 类型                                | 必选 | 校验规则 | 说明                   |
|----------------|-----------------------------------|----|------|----------------------|
| resource       | [Source](#source)                 | 是  |      | 流控规则名称来源             |
| flow           | [Flow](#flow)                     | 否  |      | flow 流量控制            |
| hotSpot        | [HotSpot](#hotspot)               | 否  |      | hot spot 热点参数流控      |
| circuitBreaker | [CircuitBreaker](#circuitbreaker) | 否  |      | circuit breaker 熔断降级 |

`flow`, `hotSpot`, `circuitBreaker` 三者至少有一项

### Source

| 名称   | 类型     | 必选 | 校验规则            | 说明                                           |
|------|--------|----|-----------------|----------------------------------------------|
| from | enum   | 否  | [HEADER, QUERY] | key 来源，可选值为 HEADER (默认), QUERY，分别表示请求头部、请求参数 |
| key  | string | 是  | min_len: 1      | k-v 对中的 key                                  |

### Flow

| 名称    | 类型                    | 必选 | 校验规则 | 说明        |
|-------|-----------------------|----|------|-----------|
| rules | [FlowRule](#flowrule) | 否  |      | flow 规则列表 |

#### FlowRule

| 名称                     | 类型                              | 必选 | 校验规则                                    | 说明                                                                                 |
|------------------------|---------------------------------|----|-----------------------------------------|------------------------------------------------------------------------------------|
| id                     | string                          | 否  |                                         | 唯一 ID                                                                              |
| resource               | string                          | 是  | min_len: 1                              | 规则名称                                                                               |
| tokenCalculateStrategy | enum                            | 否  | [DIRECT, WARMUP]                        | token 计算策略，即流控策略，可选值为 DIRECT (默认), WARMUP，分别表示直接使用 threshold 字段、使用预热方式计算 token 阈值  |
| controlBehavior        | enum                            | 否  | [REJECT, THROTTLING]                    | 流控行为，可选值为 REJECT (默认), THROTTLING，分别表示触发流控时直接拒绝请求、对请求匀速排队                          |
| threshold              | double                          | 否  |                                         | token 阈值，即流控阈值                                                                     |
| statIntervalInMs       | uint32                          | 否  |                                         | 流控统计周期，默认为 1000                                                                    |
| maxQueueingTimeMs      | uint32                          | 否  |                                         | 最大匀速排队等待时间，仅在 `controlBehavior == THROTTLING` 时生效                                  |
| relationStrategy       | enum                            | 否  | [CURRENT_RESOURCE, ASSOCIATED_RESOURCE] | 规则关联策略，可选值为 CURRENT_RESOURCE (默认), ASSOCIATED_RESOURCE，分别表示使用当前规则、关联其他 flow 规则进行流控 |
| refResource            | string                          | 否  |                                         | 关联的 flow 规则名称，仅在 `relationStrategy == ASSOCIATED_RESOURCE` 时生效                     |
| warmUpPeriodSec        | uint32                          | 否  |                                         | 预热持续时间，仅在 `tokenCalculateStrategy == WARMUP` 时生效                                   |
| warmUpColdFactor       | uint32                          | 否  |                                         | 预热因子，仅在 `tokenCalculateStrategy == WARMUP` 时生效                                     |
| blockResponse          | [BlockResponse](#blockresponse) | 否  |                                         | 流量被拦截时返回的响应消息                                                                      |

WARMUP 详见：[sentinel-golang 流量控制策略](https://sentinelguard.io/zh-cn/docs/golang/flow-control.html)

### HotSpot

| 名称          | 类型                          | 必选 | 校验规则 | 说明            |
|-------------|-----------------------------|----|------|---------------|
| rules       | [HotSpotRule](#hotspotrule) | 否  |      | hot spot 规则列表 |
| params      | string[]                    | 否  |      | 流控参数列表        |
| attachments | [Source](#source)[]         | 否  |      | 流控附件列表        |

`params`, `attachments` 两者至少有一项

#### HotSpotRule

| 名称                | 类型                              | 必选 | 校验规则                 | 说明                                                                                   |
|-------------------|---------------------------------|----|----------------------|--------------------------------------------------------------------------------------|
| id                | string                          | 否  |                      | 唯一 ID                                                                                |
| resource          | string                          | 是  | min_len: 1           | 规则名称                                                                                 |
| metricType        | enum                            | 否  | [CONCURRENCY, QPS]   | 流控统计类型，可选值为 CONCURRENCY (默认), QPS，分别表示使用并发数、请求数 QPS 作为指标进行统计                         |
| controlBehavior   | enum                            | 否  | [REJECT, THROTTLING] | 流控行为，仅在 `metricType == QPS` 时生效，可选值为 REJECT (默认), THROTTLING，分别表示触发流控时直接拒绝请求、对请求匀速排队 |
| paramIndex        | int32                           | 否  |                      | 流控参数列表下标，指定使用列表中某个参数进行流控                                                             |
| paramKey          | string                          | 否  |                      | 流控附件 Key，即 attachmentKey，指定使用某个附件进行流控，相比流控参数更灵活，因为它的 Value 来自请求头部或请求参数               |
| threshold         | int64                           | 否  |                      | 流控阈值 (针对某个流控参数/附件)                                                                   |
| durationInSec     | int64                           | 否  |                      | 流控统计周期，仅在 `metricType == QPS` 时生效                                                    |
| maxQueueingTimeMs | int64                           | 否  |                      | 最大匀速排队等待时间，仅在 `controlBehavior == THROTTLING` 时生效                                    |
| burstCount        | int64                           | 否  |                      | 静默值，仅在 `controlBehavior == REJECT` 时生效                                               |
| paramsMaxCapacity | int64                           | 否  |                      | 统计结构的容量最大值，基于 LRU 的策略，每个规则默认缓存 20000 个参数/附件的统计数据                                     |
| specificItems     | map<string, int64>              | 否  |                      | 特定参数/附件的特殊阈值配置，可以针对指定的参数/附件值单独设置限流阈值，不受 threshold 字段的限制                              |
| blockResponse     | [BlockResponse](#blockresponse) | 否  |                      | 流量被拦截时返回的响应消息                                                                        |

### CircuitBreaker

| 名称    | 类型                                        | 必选 | 校验规则 | 说明                   |
|-------|-------------------------------------------|----|------|----------------------|
| rules | [CircuitBreakerRule](#circuitbreakerrule) | 否  |      | circuit breaker 规则列表 |

#### CircuitBreakerRule

| 名称                           | 类型                              | 必选 | 校验规则                                           | 说明                                                                                                            |
|------------------------------|---------------------------------|----|------------------------------------------------|---------------------------------------------------------------------------------------------------------------|
| id                           | string                          | 否  |                                                | 唯一 ID                                                                                                         |
| resource                     | string                          | 是  | min_len: 1                                     | 该规则名称                                                                                                         |
| strategy                     | enum                            | 否  | [SLOW_REQUEST_RATIO, ERROR_RATIO, ERROR_COUNT] | 熔断策略，可选值为 SLOW_REQUEST_RATIO (默认), ERROR_RATIO, ERROR_COUNT，分别表示慢响应率、错误率、错误次数                                 |
| retryTimeoutMs               | uint32                          | 否  |                                                | 熔断打开（Open）至半打开（Half-Open）状态的持续时间，默认为 3000                                                                     |
| minRequestAmount             | uint64                          | 否  |                                                | 静默值，若当前统计周期内的请求数小于此值，即使达到熔断条件规则也不会触发                                                                          |
| statIntervalMs               | uint32                          | 否  |                                                | 统计周期，默认为 1000                                                                                                 |
| threshold                    | double                          | 否  |                                                | 熔断阈值，若 `strategy == SLOW_REQUEST_RATIO \| ERROR_RATIO`，则用小数表示百分比，否则用整数表示错误次数                                  |
| probeNum                     | uint64                          | 否  |                                                | 探测次数，熔断从半打开（Half-Open）转变为关闭（Closed）状态需要的成功请求次数                                                                |
| maxAllowedRtMs               | uint64                          | 否  |                                                | 最大允许响应时延，仅在 `strategy == SLOW_REQUEST_RATIO` 时生效                                                              |
| statSlidingWindowBucketCount | uint32                          | 否  |                                                | 统计滑动窗口的桶数量，要求 `statIntervalMs % statSlidingWindowBucketCount == 0`                                            |
| triggeredByStatusCodes       | uint32[]                        | 否  |                                                | 错误响应状态码列表，仅在 `strategy == ERROR_RATIO \| ERROR_COUNT` 时生效，默认为 \[500\]，当后端响应的状态码击中该列表中的值的次数达到 threshold 时会触发熔断 |
| blockResponse                | [BlockResponse](#blockresponse) | 否  |                                                | 流量被拦截时返回的响应                                                                                                   |

### BlockResponse

| 名称         | 类型                  | 必选 | 校验规则 | 说明                                |
|------------|---------------------|----|------|-----------------------------------|
| message    | string              | 否  |      | 响应信息，默认为 sentinel traffic control |
| statusCode | uint32              | 否  |      | 响应状态码，默认为 429                     |
| headers    | map<string, string> | 否  |      | 响应头部                              |

## 用法

假设我们有下面附加到 `localhost:10000` 的 HTTPRoute，并且有一个后端服务器监听端口 `3000`：

> 例如以下使用的后端为 [echo server](https://github.com/kubernetes-sigs/ingress-controller-conformance/tree/master/images/echoserver)

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
          port: 3000
```

### flow 流量控制

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: flow
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    sentinel:
      config:
        resource:
          from: HEADER
          key: X-Sentinel
        flow:
          rules:
            - resource: foo
              tokenCalculateStrategy: DIRECT
              controlBehavior: REJECT
              threshold: 2
              statIntervalInMs: 1000
              blockResponse:
                message: "custom msg: flow foo"
                statusCode: 503
                headers:
                  hello: "world"
```

上述流控规则表示在 1s 中内最多允许请求 2 次，多余的请求会被直接拦截并返回 503 状态码，以及自定义的 msg、头部信息

命中规则的方式是携带 `X-Sentinel: foo` 请求头部

尝试在 1s 内发送 3 次请求：

```bash
$ for ((i=0;i<3;i++)); do \
    curl -v 'http://localhost:10000/' -H 'X-Sentinel: foo'; done
# 请求 1，成功
> ...
> X-Sentinel: foo
>
< HTTP/1.1 200 OK
< ...

# 请求 2，成功
> ...
> X-Sentinel: foo
>
< HTTP/1.1 200 OK
< ...

# 请求 3，触发流控
> ...
> X-Sentinel: foo
>
< HTTP/1.1 503 Service Unavailable
< hello: world
< ...
{"msg":"custom msg: flow foo"}%
```

1s 后重新发送请求：

```bash
$ curl -I 'http://localhost:10000/' -H 'X-Sentinel: foo'
HTTP/1.1 200 OK
...
```

将头部从 `X-Sentinel: foo` 改为 `X-Sentinel: abc` 重新发送 3 次请求发现流控不生效，因为并没有配置名为 `abc` 的流控规则

### hot spot 热点参数流控

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: hot-spot
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    sentinel:
      config:
        resource:
          from: QUERY
          key: res
        hotSpot:
          attachments:
            - from: HEADER
              key: X-Header
          rules:
            - resource: bar
              metricType: QPS
              controlBehavior: REJECT
              paramKey: X-Header
              threshold: 5
              durationInSec: 1
              specificItems:
                a: 2
```

上述热点参数流控规则表示插件会从请求中获取 Key 为 `X-Header` 的头部，并结合它的值进行流控，
例如 `X-Header: a` 和 `X-Header: b` 两种请求在 1s 内都最多允许请求 5 次，
但由于 `specificItems` 字段配置了 `a: 2`，因此携带 `X-Header: a` 的请求最多只能允许请求 2 次

命中规则的方式是携带 `res=bar` 请求参数、Key 为 `X-Header` 的请求头部

尝试在 1s 内发送 3 次请求：

```bash
$ for ((i=0;i<3;i++)); do \
    curl -I 'http://localhost:10000?res=bar' -H 'X-Header: a'; done
# 请求 1，成功
HTTP/1.1 200 OK
...

# 请求 2，成功
HTTP/1.1 200 OK
...

# 请求 3，触发热点参数流控
HTTP/1.1 429 Too Many Requests
...

```

### circuit breaker 熔断降级

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: circuit-breaker
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    sentinel:
      config:
        resource:
          from: HEADER
          key: X-My-Test
        circuitBreaker:
          rules:
            - resource: baz
              strategy: ERROR_COUNT
              retryTimeoutMs: 3000
              statIntervalMs: 1000
              threshold: 5
              probeNum: 2
              triggeredByStatusCodes: [ 503 ]
              blockResponse:
                message: "custom msg: circuit breaker baz"
                statusCode: 500
```

上述熔断降级规则表示在 1s 内收到 5 次来自后端的 503 响应后会触发熔断，并且在 3s 后允许探测请求，若探测阶段收到 503 响应则会直接熔断，否则在探测成功 2 次后，熔断关闭

命中规则的方式是携带 `X-My-Test: baz` 请求头部、后端返回的状态码在配置的状态码列表中

尝试让后端返回 6 次 503 响应：

```bash
$ for ((i=0;i<6;i++)); do \
    curl -I 'http://localhost:10000/status/503' -H 'X-My-Test: baz'; done
# 请求 1，符合预期
HTTP/1.1 503 Service Unavailable
...

# ... 省略中间 3 次请求，均为 503 响应符合预期

# 请求 5，符合预期
HTTP/1.1 503 Service Unavailable
...

# 请求 6，熔断打开
HTTP/1.1 500 Internal Server Error
...
```

若开启 DEBUG 日志，会看到：

```bash
... [circuitbreaker state change] resource: baz, steategy: ErrorCount, Closed -> Open, failed times: 5
```

尝试发送 2 次成功请求后，熔断关闭：

```bash
$ curl -I 'http://localhost:10000/' -H 'X-My-Test: baz'
HTTP/1.1 200 OK
...

$ curl -I 'http://localhost:10000/' -H 'X-My-Test: baz'
HTTP/1.1 200 OK
...
```

若开启 DEBUG 日志，会看到：

```bash
... [circuitbreaker state change] resource: baz, steategy: ErrorCount, Open -> Half-Open
...
... [circuitbreaker state change] resource: baz, steategy: ErrorCount, HalfOpen -> Closed
```
