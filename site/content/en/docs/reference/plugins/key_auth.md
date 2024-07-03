---
title: Key Auth
---

## Description

The `keyAuth` plugin authenticates the client according to the consumers and the key sent in the request.

## Attribute

|       |       |
| ----- | ----- |
| Type  | Authn |
| Order | Authn |

## Configuration

| Name | Type  | Required | Validation | Description                           |
| ---- | ----- | -------- | ---------- | ------------------------------------- |
| keys | Key[] | True     | min_len: 1 | Where to find the authentication key. |

Keys configured in the `keys` field are matched one by one until one of them is matched.

### Key

| Name   | Type   | Required | Validation      | Description                                 |
|--------|--------|----------|-----------------|---------------------------------------------|
| name   | string | True     | min_len: 1      | The source's name                           |
| source | enum   | False    | [HEADER, QUERY] | Where to find the key, default to `HEADER`. |

When the `source` is `HEADER`, it fetches the key from the configured request header `name`. It can also be `QUERY`: fetch key from URL query string.

## Consumer Configuration

| Name | Type   | Required | Validation | Description        |
| ---- | ------ | -------- | ---------- | ------------------ |
| key  | string | True     | min_len: 1 | The consumer's key |

## Usage

First of all, let's create a consumer with key `rick`:

```yaml
apiVersion: htnn.mosn.io/v1
kind: Consumer
metadata:
  name: consumer
spec:
  auth:
    keyAuth:
      config:
        key: rick
```

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
kind: FilterPolicy
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
        - name: ak
          source: QUERY
```

The header `Authorization` will be checked first and then query argument `ak` will be checked after.

Let's try it out:

```
$ curl -I http://localhost:10000/ -H "Authorization: rick"
HTTP/1.1 200 OK
```

```
$ curl -I http://localhost:10000/ -H "Authorization: morty"
HTTP/1.1 401 Unauthorized
```

```
$ curl -I 'http://localhost:10000/?ak=rick'
HTTP/1.1 200 OK
```

Note that if a configured `key` exists in the request, the subsequent `key` in `keys` will not be used to authenticate the client:

```
$ curl -I 'http://localhost:10000/?ak=rick' -H "Authorization: morty"
HTTP/1.1 401 Unauthorized
```

In the example above, the request is rejected because the key in `Authorization` is incorrect. This avoids the security risk that the hacker fakes different clients by providing multiple keys.