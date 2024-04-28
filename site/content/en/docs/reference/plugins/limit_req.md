---
title: Limit Req
---

## Description

The `limitReq` plugin limits the number of requests per second to this proxy. The implementation is based on the [token bucket algorithm](https://en.wikipedia.org/wiki/Token_bucket). To draw a comparison, consider the `average` and `period` (as specified further on) to act as the refill rate for the bucket, while the `burst` represents the bucket's size.

## Attribute

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Traffic |

## Configuration
| Name    | Type                            | Required | Validation | Description                                                                                        |
|---------|---------------------------------|----------|------------|----------------------------------------------------------------------------------------------------|
| average | uint32                          | True     | gt: 0      | The threshold value, by default calculated as the number of requests per second.                   |
| period  | [Duration](../../type#duration) | False    |            | The time unit for the rate. The rate limit is defined as `average / period`. Defaults to 1 second. |
| burst   | uint32                          | False    |            | The number of requests allowed to exceed the rate. Defaults to 1.                                  |
| key     | string                          | False    |            | The key used for rate limiting. Defaults to client IP. Supports [CEL expressions](../../expr).        |

When the request rate exceeds `average / period` and the number of excess requests is over `burst`, we calculate the delay time needed to reduce the rate to the expected level. If the required delay time does not exceed the maximum delay, the request will be delayed. If the required delay time is greater than the maximum delay, the request will be dropped with a `429` HTTP status code. By default, the maximum delay is half of the rate (`1 / 2 * average / period`). If `average / period` is less than 1, it defaults to 500 milliseconds.

Requests are counted by client IP by default. You can also configure `key` to use other fields. The configuration inside `key` will be interpreted as a CEL expression. For example, `key: request.header("x-key")` means using the request header `x-key` as the dimension for rate limiting. If the value corresponding to `key` is empty, it falls back to counting by client IP. You can also provide a default value in the expression, such as `key: 'request.header("x-key") != "" ? request.header("x-key") : request.header("x-forwarded-for")'`, which means using the request header `x-key` as the dimension for rate limiting first, and if not found, then using `x-forwarded-for`.

## Usage

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
    keyAuth:
      config:
        average: 1 # limit request to 1 request per second
```

The first request will get a `200` status code, and subsequent requests will be dropped with `429`:

```
$ while true; do curl -I http://localhost:10000/ 2>/dev/null | head -1 ; done
HTTP/1.1 200 OK
HTTP/1.1 429 Too Many Requests
HTTP/1.1 429 Too Many Requests
```

If the client reduces its request rate under one request per second, all the requests won't be dropped:

```
$ while true; do curl -I http://localhost:10000/ 2>/dev/null | head -1 ; sleep 1; done
HTTP/1.1 200 OK
HTTP/1.1 200 OK
```