---
title: Consumer Restriction
---

## Description

The `consumerRestriction` plugin determines whether the current consumer has access permission based on the configuration. If there is no current consumer or the consumer does not have access permission, a 403 HTTP status code is returned.

## Attribute

|       |       |
|-------|-------|
| Type  | Authz |
| Order | Authz |

## Configuration

| Name  | Type  | Required | Validation | Description                          |
|-------|-------|----------|------------|--------------------------------------|
| allow | Rules | False    |            | List of rules allowing access access |
| deny  | Rules | False    |            | List of rules denying access access  |

Only one of `allow` or `deny` can be configured.

### Rules

| Name  | Type   | Required | Validation     | Description        |
|-------|--------|----------|----------------|--------------------|
| rules | Rule[] | True     | min_items: 1   | List of rules      |

## Rule

| Name | Type   | Required | Validation   | Description          |
|------|--------|----------|--------------|----------------------|
| name | string | True     | min_len: 1   | Name of the Consumer |


## Usage

First, let's create two consumers:

```yaml
apiVersion: htnn.mosn.io/v1
kind: Consumer
metadata:
  name: rick
spec:
  auth:
    keyAuth:
      config:
        key: rick
---
apiVersion: htnn.mosn.io/v1
kind: Consumer
metadata:
  name: doraemon
spec:
  auth:
    keyAuth:
      config:
        key: doraemon
```

Suppose we have provided the following configuration to `http://localhost:10000/time_travel`:

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
        type: Exact
        value: /time_travel
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
        keys:
        - name: Authorization
    consumerRestriction:
      config:
        allow:
          rules:
          - name: doraemon
```

`doraemon` can access `/time_travel`, while all other consumers cannot access the route.

Let's give it a try:

```
$ curl -I http://localhost:10000/time_travel -H "Authorization: doraemon"
HTTP/1.1 200 OK
$ curl -I http://localhost:10000/time_travel -H "Authorization: rick"
HTTP/1.1 403 Forbidden
```

If you want to use a deny list, replace `allow` with `deny`:

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
    keyAuth:
      config:
        keys:
        - name: Authorization
    consumerRestriction:
      config:
        deny:
          rules:
          - name: rick
```
