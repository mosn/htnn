---
title: Demo
---

## Description

The `demo` plugin is used to show how to add a plugin to htnn.

## Attribute

|       |             |
|-------|-------------|
| Type  | General     |
| Order | Unspecified |

## Configuration

| Name     | Type   | Required | Validation | Description                                                                      |
|----------|--------|----------|------------|----------------------------------------------------------------------------------|
| hostName | string | True     | min_len: 1 | The request header name which will contain `hello, ...` greeting to the upstream |

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

By applying the configuration below, this plugin will insert a header `John Doe: hello, $guest_name` in the request. The value of `$guest_name` is determined by the value of filter state name `guest_name`.

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
    demo:
      config:
        hostName: "John Doe"
```
