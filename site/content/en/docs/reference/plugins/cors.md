---
title: CORS
---

## Description

The `cors` plugin handles Cross-Origin Resource Sharing requests by leveraging Envoy's `cors` filter.

## Attribute

|        |              |
|--------|--------------|
| Type   | Security     |
| Order  | Outer        |
| Status | Experimental |

## Configuration

See the corresponding [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/http/http_filters/cors_filter).

## Usage

Assuming we have the following HTTPRoute attached to `localhost:10000`, with a backend server listening on port `8080`:

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

By applying the configuration below, Cross-Origin Resource Sharing requests sent to `http://localhost:10000/` will be processed. If the `Origin` header of an OPTIONS request matches the regular expression `.*\.default\.local`, then the corresponding response will include the configured `Access-Control-Allow-*` response headers.

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
    cors:
      config:
        allowOriginStringMatch:
        - safeRegex:
            regex: ".*\.default\.local"
        allowMethods: POST
```

Let's try it out:

```shell
$ curl -X OPTIONS http://localhost:10000/ -H "Origin: https://x.efault.local" -H "Access-Control-Request-Method: GET" -i
HTTP/1.1 200 OK
server: istio-envoy
...

$ curl -X OPTIONS http://localhost:10000/ -H "Origin: https://x.default.local" -H "Access-Control-Request-Method: GET" -i
HTTP/1.1 200 OK
access-control-allow-origin: https://x.default.local
access-control-allow-methods: POST
server: istio-envoy
...
```
