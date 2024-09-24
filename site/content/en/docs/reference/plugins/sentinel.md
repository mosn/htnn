---
title: Sentinel
---

## Description

The `sentinel` plugin utilizes [sentinel-golang](https://github.com/alibaba/sentinel-golang) to provide traffic control capabilities. Currently, it supports the following three configuration rules:

- flow: Traffic control
- hot spot: Hot spot parameter traffic control
- circuit breaker: Circuit breaker and degradation

## Attribute

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Traffic |

## Configuration

The configuration fields are largely consistent with sentinel-golang v1.0.4, with some non-pluggable features removed.

For detailed configuration instructions, please refer to the [sentinel-golang official documentation](https://sentinelguard.io/zh-cn/docs/golang/quick-start.html).

| Name           | Type                              | Required | Validation | Description                         |
|----------------|-----------------------------------|----------|------------|-------------------------------------|
| resource       | [Source](#source)                 | True     |            | Source of traffic control rule name |
| flow           | [Flow](#flow)                     | False    |            | Flow traffic control                |
| hotSpot        | [HotSpot](#hotspot)               | False    |            | Hot spot traffic control            |
| circuitBreaker | [CircuitBreaker](#circuitbreaker) | False    |            | Circuit breaker and degradation     |

At least one of `flow`, `hotSpot`, or `circuitBreaker` must be provided.

### Source

| Name | Type   | Required | Validation      | Description                                                                                      |
|------|--------|----------|-----------------|--------------------------------------------------------------------------------------------------|
| from | enum   | False    | [HEADER, QUERY] | Key source, options are HEADER (default), QUERY, indicating request header or request parameters |
| key  | string | True     | min_len: 1      | Key in the key-value pair                                                                        |

### Flow

| Name  | Type                  | Required | Validation | Description        |
|-------|-----------------------|----------|------------|--------------------|
| rules | [FlowRule](#flowrule) | False    |            | List of flow rules |

#### FlowRule

| Name                   | Type                            | Required | Validation                              | Description                                                                                                      |
|------------------------|---------------------------------|----------|-----------------------------------------|------------------------------------------------------------------------------------------------------------------|
| id                     | string                          | False    |                                         | Unique ID                                                                                                        |
| resource               | string                          | True     | min_len: 1                              | Rule name                                                                                                        |
| tokenCalculateStrategy | enum                            | False    | [DIRECT, WARMUP]                        | Token calculation strategy, options are DIRECT (default) and WARMUP, determining how the threshold is calculated |
| controlBehavior        | enum                            | False    | [REJECT, THROTTLING]                    | Control behavior, options are REJECT (default) and THROTTLING, defining whether to reject or throttle requests   |
| threshold              | double                          | False    |                                         | Token threshold, i.e., the traffic control threshold                                                             |
| statIntervalInMs       | uint32                          | False    |                                         | Traffic control statistical interval, default is 1000 ms                                                         |
| maxQueueingTimeMs      | uint32                          | False    |                                         | Maximum queueing time, effective only when `controlBehavior == THROTTLING`                                       |
| relationStrategy       | enum                            | False    | [CURRENT_RESOURCE, ASSOCIATED_RESOURCE] | Rule relation strategy, options are CURRENT_RESOURCE (default) and ASSOCIATED_RESOURCE                           |
| refResource            | string                          | False    |                                         | Associated flow rule name, effective only when `relationStrategy == ASSOCIATED_RESOURCE`                         |
| warmUpPeriodSec        | uint32                          | False    |                                         | Warm-up duration, effective only when `tokenCalculateStrategy == WARMUP`                                         |
| warmUpColdFactor       | uint32                          | False    |                                         | Warm-up factor, effective only when `tokenCalculateStrategy == WARMUP`                                           |
| blockResponse          | [BlockResponse](#blockresponse) | False    |                                         | Response message when traffic is blocked                                                                         |

For more information on WARMUP, see: [sentinel-golang flow control strategy](https://sentinelguard.io/zh-cn/docs/golang/flow-control.html).

### HotSpot

| Name        | Type                        | Required | Validation | Description                         |
|-------------|-----------------------------|----------|------------|-------------------------------------|
| rules       | [HotSpotRule](#hotspotrule) | False    |            | List of hot spot rules              |
| params      | string[]                    | False    |            | List of traffic control parameters  |
| attachments | [Source](#source)[]         | False    |            | List of traffic control attachments |

At least one of `params` or `attachments` must be provided.

#### HotSpotRule

| Name              | Type                            | Required | Validation           | Description                                                                                                              |
|-------------------|---------------------------------|----------|----------------------|--------------------------------------------------------------------------------------------------------------------------|
| id                | string                          | False    |                      | Unique ID                                                                                                                |
| resource          | string                          | True     | min_len: 1           | Rule name                                                                                                                |
| metricType        | enum                            | False    | [CONCURRENCY, QPS]   | Traffic control metric type, options are CONCURRENCY (default) and QPS                                                   |
| controlBehavior   | enum                            | False    | [REJECT, THROTTLING] | Control behavior, effective only when `metricType == QPS`, options are REJECT (default) and THROTTLING                   |
| paramIndex        | int32                           | False    |                      | Index of the traffic control parameter list, specifying which parameter to control                                       |
| paramKey          | string                          | False    |                      | Key for traffic control attachment, specifying which attachment to control                                               |
| threshold         | int64                           | False    |                      | Traffic control threshold (for a specific parameter/attachment)                                                          |
| durationInSec     | int64                           | False    |                      | Traffic control statistical interval, effective only when `metricType == QPS`                                            |
| maxQueueingTimeMs | int64                           | False    |                      | Maximum queueing time, effective only when `controlBehavior == THROTTLING`                                               |
| burstCount        | int64                           | False    |                      | Burst value, effective only when `controlBehavior == REJECT`                                                             |
| paramsMaxCapacity | int64                           | False    |                      | Maximum capacity for the statistical structure, based on LRU, with each rule caching up to 20,000 parameters/attachments |
| specificItems     | map<string, int64>              | False    |                      | Special threshold settings for specific parameters/attachments                                                           |
| blockResponse     | [BlockResponse](#blockresponse) | False    |                      | Response message when traffic is blocked                                                                                 |

### CircuitBreaker

| Name    | Type                                        | Required | Validation | Description          |
|--------|---------------------------------------------|----------|------------|----------------------|
| rules  | [CircuitBreakerRule](#circuitbreakerrule)    | False    |            | List of circuit breaker rules |

#### CircuitBreakerRule

| Name                         | Type                            | Required | Validation                                     | Description                                                                                                                      |
|------------------------------|---------------------------------|----------|------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------|
| id                           | string                          | False    |                                                | Unique ID                                                                                                                        |
| resource                     | string                          | True     | min_len: 1                                     | Rule name                                                                                                                        |
| strategy                     | enum                            | False    | [SLOW_REQUEST_RATIO, ERROR_RATIO, ERROR_COUNT] | Circuit breaker strategy, options are SLOW_REQUEST_RATIO (default), ERROR_RATIO, and ERROR_COUNT                                 |
| retryTimeoutMs               | uint32                          | False    |                                                | Duration from open to half-open state, default is 3000 ms                                                                        |
| minRequestAmount             | uint64                          | False    |                                                | Silent value, circuit breaker will not trigger if requests in the current statistical cycle are below this amount                |
| statIntervalMs               | uint32                          | False    |                                                | Statistical interval, default is 1000 ms                                                                                         |
| threshold                    | double                          | False    |                                                | Circuit breaker threshold, expressed as a percentage for `SLOW_REQUEST_RATIO` and `ERROR_RATIO`, or as a count for `ERROR_COUNT` |
| probeNum                     | uint64                          | False    |                                                | Number of successful probes needed to close the circuit breaker                                                                  |
| maxAllowedRtMs               | uint64                          | False    |                                                | Maximum allowed response time, effective only when `strategy == SLOW_REQUEST_RATIO`                                              |
| statSlidingWindowBucketCount | uint32                          | False    |                                                | Number of buckets in the sliding window, must satisfy `statIntervalMs % statSlidingWindowBucketCount == 0`                       |
| triggeredByStatusCodes       | uint32[]                        | False    |                                                | List of error status codes, effective only when `strategy == ERROR_RATIO \| ERROR_COUNT`, default is \[500\]                     |
| blockResponse                | [BlockResponse](#blockresponse) | False    |                                                | Response message when traffic is blocked                                                                                         |

### BlockResponse

| Name       | Type                | Required | Validation | Description                                             |
|------------|---------------------|----------|------------|---------------------------------------------------------|
| message    | string              | False    |            | Response message, default is "sentinel traffic control" |
| statusCode | uint32              | False    |            | Response status code, default is 429                    |
| headers    | map<string, string> | False    |            | Response headers                                        |

## Usage

Suppose we have the following HTTPRoute attached to `localhost:10000` with a backend server listening on port `3000`:

> For example, the backend used here is the [echo server](https://github.com/kubernetes-sigs/ingress-controller-conformance/tree/master/images/echoserver).
>
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

### flow traffic control

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

The above flow control rule indicates that a maximum of 2 requests are allowed within 1 second. Any additional requests will be directly intercepted and returned with a 503 status code, along with a custom message and header information.

The rule is triggered by sending requests with the `X-Sentinel: foo` request header.

Try sending 3 requests within 1 second:

```bash
$ for ((i=0;i<3;i++)); do \
    curl -v 'http://localhost:10000/' -H 'X-Sentinel: foo'; done
# Request 1, success
> ...
> X-Sentinel: foo
>
< HTTP/1.1 200 OK
< ...

# Request 2, success
> ...
> X-Sentinel: foo
>
< HTTP/1.1 200 OK
< ...

# Request 3, flow control triggered
> ...
> X-Sentinel: foo
>
< HTTP/1.1 503 Service Unavailable
< hello: world
< ...
{"msg":"custom msg: flow foo"}%
```

Send a request again after 1 second:

```bash
$ curl -I 'http://localhost:10000/' -H 'X-Sentinel: foo'
HTTP/1.1 200 OK
...
```

Change the header from `X-Sentinel: foo` to `X-Sentinel: abc` and send 3 requests again, finding that the flow control rule is not triggered because there is no configured flow control rule named `abc`.

### hot spot parameter flow control

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

The above hot spot parameter flow control rule indicates that the plugin will extract the `X-Header` request header and use its value for flow control.
For example, requests with `X-Header: a` and `X-Header: b` are allowed up to 5 requests per second,
but since the `specificItems` field is configured with `a: 2`, requests with `X-Header: a` are only allowed up to 2 requests.

The rule is triggered by sending requests with the query parameter `res=bar` and the `X-Header` request header.

Try sending 3 requests within 1 second:

```bash
$ for ((i=0;i<3;i++)); do \
    curl -I 'http://localhost:10000?res=bar' -H 'X-Header: a'; done
# Request 1, success
HTTP/1.1 200 OK
...

# Request 2, success
HTTP/1.1 200 OK
...

# Request 3, hot spot parameter flow control triggered
HTTP/1.1 429 Too Many Requests
...

```

### circuit breaker degradation

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

The above circuit breaker degradation rule indicates that when 5 requests receive a 503 response from the backend within 1 second,
the circuit breaker will be triggered. After 3 seconds, probe requests are allowed. If a probe request receives a 503 response, the circuit breaker will immediately open again. Otherwise, after 2 successful probe requests, the circuit breaker will close.

The rule is triggered by sending requests with the `X-My-Test: baz` request header and receiving status codes that match those in the configured list.

Try making the backend return 503 responses 6 times:

```bash
$ for ((i=0;i<6;i++)); do \
    curl -I 'http://localhost:10000/status/503' -H 'X-My-Test: baz'; done
# Request 1, expected behavior
HTTP/1.1 503 Service Unavailable
...

# ... skipping 3 middle requests, all returning 503 as expected

# Request 5, expected behavior
HTTP/1.1 503 Service Unavailable
...

# Request 6, circuit breaker opens
HTTP/1.1 500 Internal Server Error
...
```

If DEBUG logging is enabled, you will see:

```bash
... [circuitbreaker state change] resource: baz, steategy: ErrorCount, Closed -> Open, failed times: 5
```

Try sending 2 successful requests after, and the circuit breaker will close:

```bash
$ curl -I 'http://localhost:10000/' -H 'X-My-Test: baz'
HTTP/1.1 200 OK
...

$ curl -I 'http://localhost:10000/' -H 'X-My-Test: baz'
HTTP/1.1 200 OK
...
```

If DEBUG logging is enabled, you will see:

```bash
... [circuitbreaker state change] resource: baz, steategy: ErrorCount, Open -> Half-Open
...
... [circuitbreaker state change] resource: baz, steategy: ErrorCount, HalfOpen -> Closed
```
