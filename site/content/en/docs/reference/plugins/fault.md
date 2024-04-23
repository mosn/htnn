---
title: Fault
---

## Description

The `fault` plugin supports response and delay injections by leveraging Envoy's `fault` filter.

## Attribute

|       |         |
|-------|---------|
| Type  | General |
| Order | Outer   |

## Configuration

See the corresponding [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.29.4/configuration/http/http_filters/fault_filter).

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
    namespace: default
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend
      port: 8080
```

By applying the configuration below, requests sent to `http://localhost:10000/` will receive a 401 response 100% of the time:

```yaml
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
    fault:
      config:
        abort:
          http_status: 401
          percentage:
            numerator: 100
```

Let's try it out:

```
$ curl http://localhost:10000/ -i 2>/dev/null | head -1
HTTP/1.1 401 Unauthorized
```
