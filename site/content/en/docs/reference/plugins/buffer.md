---
title: Buffer
---

## Description

The `buffer` plugin fully buffers the complete request, by leveraging Envoy's `buffer` filter. It can be used for several purposes, like:

* Protecting applications from high network latency.
* Some plugins will buffer the whole request, which may trigger a 413 HTTP status code when the max request body size is met. We can set the `maxRequestBytes` larger than `per_connection_buffer_limit_bytes` to increase the max request body size per route.

## Attribute

|       |         |
|-------|---------|
| Type  | General |
| Order | Outer   |

## Configuration

| Name            | Type   | Required | Validation | Description                                                                                |
|-----------------|--------|----------|------------|--------------------------------------------------------------------------------------------|
| maxRequestBytes | uint32 | True     |            | The maximum request body size that the filter will buffer before returning a 413 response. |

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
```

By applying the configuration below, the request to `http://localhost:10000/` is buffered until the body length meets the max requests bytes 4:

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

We can test it out:

```shell
$ curl -d "hello" http://localhost:10000/ -i
HTTP/1.1 413 Payload Too Large
```

```shell
$ curl -d "hell" http://localhost:10000/ -i
HTTP/1.1 200 OK
```
