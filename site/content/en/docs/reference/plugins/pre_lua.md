---
title: Pre Lua
---

## Description

This plugin allows Lua snippets to be run at the beginning and the end of a request, by leveraging Envoy's `lua` filter.

Because Envoy uses the onion model to proxy requests, the execution order is:

1. request starts
2. running pre-lua and other plugins in `Pre` group
3. running other plugins
4. proxy to the upstream
5. running other plugins with the response
6. running pre-lua and other plugins in `Pre` group, with the response
7. request ends

## Attribute

|       |         |
|-------|---------|
| Type  | General |
| Order | Pre     |

## Configuration

See the corresponding [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.28.0/configuration/http/http_filters/lua_filter).

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

By applying the configuration below, we can interrupt the requests to `http://localhost:10000/` with the custom response:

```yaml
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    pre_lua:
      config:
        source_code:
          inline_string: |
            function envoy_on_request(handle)
              local resp_headers = {[":status"] = "200"}
              local data = "hello, world"
              handle:respond(
                resp_headers,
                data
              )
            end
```

We can test it out:

```
$ curl http://localhost:10000/
HTTP/1.1 200 OK
content-length: 12
...

hello, world
```