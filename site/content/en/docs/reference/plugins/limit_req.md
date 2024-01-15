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

| Name    | Type                            | Required | Validation | Description                                                                                                             |
|---------|---------------------------------|----------|------------|-------------------------------------------------------------------------------------------------------------------------|
| average | int32                           | True     | gt: 0      | The threshold, by default in requests per second                                                                        |
| period  | [Duration](../../type#duration) | False    |            | The time unit of rate. The limit rate is defined as `average / period`. Default to 1s, which means requests per second. |
| burst   | int32                           | False    | gt: 0      | The number of requests allowed to exceed the rate. Default to 1.                                                        |

The number of requests is counted per client IP. When the request rate exceeds the `average / period` rate, and the number of exceeded requests is more than `burst`, we will calculate a required delay time to slow down the rate to the expectation. The request is delayed if the required delay time is not larger than the maximum delay. If the required delay is larger than the maximum delay, the request is dropped with a `429` HTTP status code.

The maximum delay is half of the rate (`1 / 2 * average / period`) by default, and 500ms if the `average / period` is less than 1.

## Usage

Assumed we provide a configuration to `http://localhost:10000/` like:

```yaml
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