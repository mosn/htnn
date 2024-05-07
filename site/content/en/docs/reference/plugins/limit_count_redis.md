---
title: Limit Count Redis
---

## Description

The `limitCountRedis` plugin implements a global fixed window rate-limiting by storing the count statistics in Redis. Users can control the number of client accesses within a given time for different dimensions using this plugin.

## Attribute

|       |         |
| ----- | ------- |
| Type  | Traffic |
| Order | Traffic |

## Configuration

| Name                    | Type                                | Required | Validation                 | Description                                                                                                                        |
| ----------------------- | ----------------------------------- | -------- | -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| address                 | string                              | True     |                            | Redis address                                                                                                                      |
| cluster                 | Cluster                             | True     |                            | Redis cluster configuration. Only one of `address` and `cluster` can be configured.                                                |
| rules                   | Rule                                | True     | min_items: 1, max_items: 8 | Rules                                                                                                                              |
| failureModeDeny         | boolean                             | False    |                            | By default, if access to Redis fails, the request is allowed through. When true, it denies the request.                            |
| enableLimitQuotaHeaders | boolean                             | False    |                            | Whether to set response headers related to rate-limiting quotas                                                                    |
| username                | string                              | False    |                            | Username for accessing Redis                                                                                                       |
| password                | string                              | False    |                            | Password for accessing Redis                                                                                                       |
| tls                     | boolean                             | False    |                            | Whether to access Redis over TLS                                                                                                   |
| tlsSkipVerify           | boolean                             | False    |                            | Whether to skip verification when accessing Redis over TLS                                                                         |
| statusOnError           | [StatusCode](../../type#statuscode) | False    |                            | The status code used to deny requests when Redis is inaccessible and `failureModeDeny` is true. Defaults to 500.                   |
| rateLimitedStatus       | [StatusCode](../../type#statuscode) | False    |                            | The status code for responses denied due to rate-limiting. Defaults to 429. This setting only takes effect when it's 400 or above. |

Each rule's count is independent. Rate-limiting action is triggered once any rule's quota is exhausted. Responses that are denied due to rate-limiting will include the header `x-envoy-ratelimited: true`. If `enableLimitQuotaHeaders` is set to `true` and accessing to redis succeed, all responses will include the following three headers:

* `x-ratelimit-limit`: Represents the applied rate-limiting rule. The format is "the rule with the least remaining quota, (rule quota;w=time window){one or more rules}", e.g., `2, 2;w=60`.
* `x-ratelimit-remaining`: Represents the remaining quota of the rule with the least remaining quota, with a minimum value of `0`.
* `x-ratelimit-reset`: Represents when the rule with the least remaining quota will reset, in seconds, e.g., `59`. Note that due to network latency and other factors, this value is not precise.

### Cluster

| Name      | Type     | Required | Validation   | Description   |
| --------- | -------- | -------- | ------------ | ------------- |
| addresses | string[] | True     | min_items: 1 | Redis address |

### Rule

| Name       | Type                            | Required | Validation | Description                                                                                    |
| ---------- | ------------------------------- | -------- | ---------- | ---------------------------------------------------------------------------------------------- |
| timeWindow | [Duration](../../type#duration) | True     | >= 1s      | Time window                                                                                    |
| count      | uint32                          | True     | >= 1       | Count                                                                                          |
| key        | string                          | False    |            | The key used for rate limiting. Defaults to client IP. Supports [CEL expressions](../../expr). |

Requests are counted by client IP by default. You can also configure `key` to use other fields. The configuration inside `key` will be interpreted as a CEL expression. For example, `key: request.header("x-key")` means using the request header `x-key` as the dimension for rate limiting. If the value corresponding to `key` is empty, it falls back to counting by client IP.

## Usage

First, let's assume we have a Redis service `redis.service` which is listening on port 6379.

Assumed we have the HTTPRoute below attached to `localhost:10000`, and a backend server listening to port `8080`:

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
        address: "redis.service:6379"
        enableLimitQuotaHeaders: true
        failureModeDeny: true
        rules:
        - count: 2
          timeWindow: "60s"
```

When making requests to the `/echo` path, the first two requests will succeed, and subsequent requests will fail:

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

After one minute, new requests will succeed:

```
$ curl http://localhost:10000/echo -i
HTTP/1.1 200 OK
x-ratelimit-limit: 2, 2;w=60
x-ratelimit-remaining: 1
x-ratelimit-reset: 60
date: Tue, 20 Feb 2024 03:54:16 GMT
```
