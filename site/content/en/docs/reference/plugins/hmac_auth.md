---
title: HMAC Auth
---

## Description

The `hmacAuth` plugin authenticates the client based on the consumer configuration and the signature sent in the request, using the [HMAC](https://en.wikipedia.org/wiki/HMAC) algorithm. Given the significant differences in the implementation of HMAC Auth signatures by different gateways and service providers, it is unlikely to flexibly support multiple signature methods through configuration. This plugin aims to provide the same [signature method](https://apisix.apache.org/docs/apisix/plugins/hmac-auth/#example-usage) as Apache APISIX. You can use this plugin as an example to develop your own HMAC Auth plugin.

## Attribute

|       |       |
|-------|-------|
| Type  | Authn |
| Order | Authn |

## Configuration

| Name            | Type   | Required | Validation | Description                                                                                                                            |
|-----------------|--------|----------|------------|----------------------------------------------------------------------------------------------------------------------------------------|
| signatureHeader | string | False    |            | The request header that contains the signature. Default is `x-hmac-signature`                                                          |
| accessKeyHeader | string | False    |            | The request header that contains the Access Key. Default is `x-hmac-access-key`                                                        |
| dateHeader      | string | False    |            | The request header that contains the timestamp. Default is `date`. The timestamp format is GMT, such as `Fri Jan  5 16:10:54 CST 2024` |

If the configured `accessKeyHeader` is not present, no consumer will be matched.
If the configured `signatureHeader` is not present, the signature in the request will be deemed as an empty string.
If the configured `dateHeader` is not present, the timestamp will be deemed as an empty string.

## Consumer Configuration

| Name          | Type     | Required | Validation                              | Description                                                                                                               |
|---------------|----------|----------|-----------------------------------------|---------------------------------------------------------------------------------------------------------------------------|
| accessKey     | string   | True     | min_len: 1                              | The consumer's access key.                                                                                                |
| secretKey     | string   | True     | min_len: 1                              | The consumer's secret key.                                                                                                |
| algorithm     | enum     | False    | [HMAC_SHA256, HMAC_SHA384, HMAC_SHA512] | The algorithm. Default is `HMAC_SHA256`.                                                                                  |
| signedHeaders | string[] | False    | items.string.min_len = 1                | The list of request header names used to form the signature. Note the case sensitivity must match actual request headers. |

## Usage

First, let's create a consumer:

```yaml
apiVersion: htnn.mosn.io/v1
kind: Consumer
metadata:
  name: consumer
spec:
  auth:
    hmacAuth:
      config:
        accessKey: ak
        secretKey: sk
        signedHeaders:
        - x-custom-a
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
    hmacAuth:
      config:
        signatureHeader: x-sign-hdr
        accessKeyHeader: x-ak
```

The plugin will read the access key from the request header `x-ak`, the signature from `x-sign-hdr`, the timestamp from `date`, and additional signed data from `x-custom-a`.

The signature algorithm of `hmacAuth` is consistent with Apache APISIX, details can be found in [APISIX signature algorithm details](https://apisix.apache.org/docs/apisix/plugins/hmac-auth/#example-usage).

Let's give it a try:

```
$ curl -I 'http://localhost:10000/echo?age=36&address=&title=ops&title=dev' -H "x-ak: ak" \
    -H "x-sign-hdr: E6m5y84WIu/XeeIox2VZes/+xd/8QPRSMKqo+lp3cAo=" \
    -H "date: Fri Jan  5 16:10:54 CST 2024" -H "x-custom-a: test"
HTTP/1.1 200 OK
```

A slight change in the signature yields a different result:

```
$ curl -I 'http://localhost:10000/echo?age=36&address=&title=ops&title=dev' -H "x-ak: ak" \
    -H "x-sign-hdr: E6m5y84WIu/XeeIox2VZea/+xd/8QPRSMKqo+lp3cAo=" \
    -H "date: Fri Jan  5 16:10:54 CST 2024" -H "x-custom-a: test"
HTTP/1.1 401 Unauthorized
```
