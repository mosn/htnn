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

### Filter Configuration

| Name | Type  | Required | Validation | Description                           |
| ---- | ----- | -------- | ---------- | ------------------------------------- |
| keys | Key[] | True     | min_len: 1 | Where to find the authentication key. |

Keys configured in the `keys` field are matched one by one until one of them is matched.

#### Key

| Name   | Type   | Required | Validation      | Description                                   |
| ------ | ------ | -------- | --------------- | --------------------------------------------- |
| name   | string | True     | min_len: 1      | The source's name                             |
| source | enum   | False    | [header, query] | Where to find the token, default to `header`. |

When the `source` is `header`, it fetches the token from the configured request header `name`. It can also be `query`: fetch token from URL query string.

### Consumer Configuration

| Name | Type   | Required | Validation | Description        |
| ---- | ------ | -------- | ---------- | ------------------ |
| key  | string | True     | min_len: 1 | The consumer's key |

## Usage

First of all, let's create a consumer with key `rick`:

```yaml
apiVersion: mosn.io/v1
kind: Consumer
metadata:
  name: consumer
  namespace: default
spec:
  auth:
    keyAuth:
      config:
        key: rick
```

Assumed we provide a configuration to `http://127.0.0.1:10000/` like:

```yaml
keys:
  - name: Authorization
  - name: ak
    source: query
```

The header `Authorization` will be checked first and then query argument `ak` will be checked after.

Let's try it out:

```
$ curl -I http://127.0.0.1:10000/ -H "Authorization: rick"
HTTP/1.1 200 OK
```

```
$ curl -I http://127.0.0.1:10000/ -H "Authorization: morty"
HTTP/1.1 401 Unauthorized
```

```
$ curl -I 'http://127.0.0.1:10000/?ak=rick'
HTTP/1.1 200 OK
```

Note that if a `key` exists in the request, the subsequent keys will not be used to authenticate the client:

```
$ curl -I 'http://127.0.0.1:10000/?ak=rick' -H "Authorization: morty"
HTTP/1.1 401 Unauthorized
```

In the example above, the request is rejected because the key in `Authorization` is incorrect. This avoids the security risk that the hacker fakes different clients by providing multiple keys.