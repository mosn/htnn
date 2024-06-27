---
title: CEL Script
---

## Description

The `celScript` plugin determines whether the current request can access the upstream by executing the user-configured [CEL expressions](../../expr). Compared to static Go code, CEL expressions allow dynamic runtime configuration. Compared to complex rule files, CEL expressions execute faster. Compared to Lua scripts, CEL expressions run in a sandboxed environment, which is more secure.

## Attribute

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Traffic |

## Configuration

| Name    | Type   | Required | Validation | Description                                                                 |
|---------|--------|----------|------------|-----------------------------------------------------------------------------|
| allowIf | string | False    |            | The expression to control access. If the expression evaluates to false, a 403 HTTP status code is returned |

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

Let's apply the configuration below:

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
    celScript:
      config:
        allowIf: 'request.path() == "/echo" && request.method() == "GET"'
```

The `allowIf` expression requires that the request path is `/echo`, and the method is `GET`.

Sending a GET request to the `/echo` path will succeed:

```
$ curl http://localhost:10000/echo
HTTP/1.1 200 OK
```

Sending a POST request to the `/echo` path will fail:

```
$ curl -X POST http://localhost:10000/echo
HTTP/1.1 403 Forbidden
```
