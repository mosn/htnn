---
title: HMAC Auth
---

## 说明

`hmacAuth` 插件根据消费者配置和请求中发送的签名，通过 [HMAC](https://zh.wikipedia.org/wiki/HMAC) 算法对客户端进行认证。

由于不同网关、不同服务提供商实现的 HMAC Auth 签名方式不同，而且它们之间的差别很大，不太可能通过配置灵活支持多种签名方式。本插件旨在提供和 Apache APISIX 一样的[签名方式](https://apisix.apache.org/zh/docs/apisix/plugins/hmac-auth/#%E7%AD%BE%E5%90%8D%E7%AE%97%E6%B3%95%E8%AF%A6%E8%A7%A3)。你可以把该插件作为示例，开发自己的 HMAC Auth 插件。

## 属性

|       |       |
|-------|-------|
| Type  | Authn |
| Order | Authn |

## 配置

| 名称            | 类型   | 必选 | 校验规则 | 说明                                                                                     |
|-----------------|--------|------|----------|------------------------------------------------------------------------------------------|
| signatureHeader | string | 否   |          | 包含签名的请求头。默认为 `x-hmac-signature`                                              |
| accessKeyHeader | string | 否   |          | 包含 Access Key 的请求头。默认为 `x-hmac-access-key`                                     |
| dateHeader      | string | 否   |          | 包含时间戳的请求头。默认为 `date`。时间戳的格式为 GMT，如 `Fri Jan  5 16:10:54 CST 2024` |

如果配置的 `accessKeyHeader` 不存在，则不会匹配任何消费者。
如果配置的 `signatureHeader` 不存在，则视作请求中的签名为空字符串。
如果配置的 `dateHeader` 不存在，则视作时间戳为空字符串。

## 消费者配置

| 名称          | 类型     | 必选 | 校验规则                                | 说明                                                                 |
|---------------|----------|------|-----------------------------------------|----------------------------------------------------------------------|
| accessKey     | string   | 是   | min_len: 1                              | 消费者的 access key                                                  |
| secretKey     | string   | 是   | min_len: 1                              | 消费者的 secret key                                                  |
| algorithm     | enum     | 否   | [HMAC_SHA256, HMAC_SHA384, HMAC_SHA512] | 算法。默认为 `HMAC_SHA256`。                                         |
| signedHeaders | string[] | 否   | items.string.min_len = 1                | 用于构成签名的请求头名称列表。注意这里需要和实际的请求头大小写一致。 |

## 用法

首先，让我们创建一个消费者：

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

假设我们有下面附加到 `localhost:10000` 的 HTTPRoute，并且有一个后端服务器监听端口 `8080`：

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

让我们应用下面的配置：

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
    hmacAuth:
      config:
        signatureHeader: x-sign-hdr
        accessKeyHeader: x-ak
```

插件将从请求头 `x-ak` 中读取 access key，从请求头 `x-sign-hdr` 中读取签名，从请求头 `date` 中读取时间戳，从请求头 `x-custom-a` 中读取额外的已签名数据。

`hmacAuth` 的签名算法和 Apache APISIX 是一致的，具体见 [APISIX 的签名算法详解](https://apisix.apache.org/zh/docs/apisix/plugins/hmac-auth/#%E7%AD%BE%E5%90%8D%E7%AE%97%E6%B3%95%E8%AF%A6%E8%A7%A3)。

让我们试一试：

```
$ curl -I 'http://localhost:10000/echo?age=36&address=&title=ops&title=dev' -H "x-ak: ak" \
    -H "x-sign-hdr: E6m5y84WIu/XeeIox2VZes/+xd/8QPRSMKqo+lp3cAo=" \
    -H "date: Fri Jan  5 16:10:54 CST 2024" -H "x-custom-a: test"
HTTP/1.1 200 OK
```

稍微改变下签名，会有不一样的结果：

```
$ curl -I 'http://localhost:10000/echo?age=36&address=&title=ops&title=dev' -H "x-ak: ak" \
    -H "x-sign-hdr: E6m5y84WIu/XeeIox2VZea/+xd/8QPRSMKqo+lp3cAo=" \
    -H "date: Fri Jan  5 16:10:54 CST 2024" -H "x-custom-a: test"
HTTP/1.1 401 Unauthorized
```
