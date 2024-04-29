---
title: Key Auth
---

## 说明

`keyAuth` 插件根据消费者配置和请求中发送的密钥对客户端进行认证。

## 属性

|       |       |
|-------|-------|
| Type  | Authn |
| Order | Authn |

## 配置

| 名称 | 类型  | 必选 | 校验规则   | 说明                 |
|------|-------|------|------------|----------------------|
| keys | Key[] | 是   | min_len: 1 | 查找认证密钥的位置。 |

在 `keys` 字段中配置的密钥将逐一匹配，直到找到一个匹配的密钥。

### Key

| 名称   | 类型   | 必选 | 校验规则        | 说明                              |
|--------|--------|------|-----------------|-----------------------------------|
| name   | string | 是   | min_len: 1      | 来源的名称                        |
| source | enum   | 否   | [HEADER, QUERY] | 查找密钥的位置，默认为 `HEADER`。 |

当 `source` 是 `HEADER` 时，它会从配置的请求头 `name` 中获取密钥。它也可以是 `QUERY`：此时会从 URL 查询字符串中获取密钥。

## 消费者配置

| 名称 | 类型   | 必选 | 校验规则   | 说明           |
|------|--------|------|------------|----------------|
| key  | string | 是   | min_len: 1 | 消费者的密钥。 |

## 用法

首先，让我们创建一个带有密钥 `rick` 的消费者：

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
        - name: ak
          source: QUERY
```

插件将首先检查请求头 `Authorization`，然后检查查询参数 `ak`。

让我们试一试：

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

注意，如果请求中存在一个配置的 `key`，那么在 `keys` 中后续的 `key` 将不会用于认证客户端：

```
$ curl -I 'http://localhost:10000/?ak=rick' -H "Authorization: morty"
HTTP/1.1 401 Unauthorized
```

在上面的例子中，请求被拒绝，因为 `Authorization` 中的密钥不正确。这避免了黑客通过提供多个密钥伪造不同客户端的安全风险。