---
title: OPA
---

## 说明

`opa` 插件集成了 [Open Policy Agent (OPA)](https://www.openpolicyagent.org)。
您可以用它与远程 OPA 服务交互（远程模式），或通过本地策略代码对请求鉴权（本地模式）。

## 属性

|       |       |
| ----- | ----- |
| Type  | Authz |
| Order | Authz |

## 配置

| 名称   | 类型   | 必选 | 校验规则        | 说明                                       |
|--------|--------|------|-----------------|-------------------------------------------|
| remote | Remote | 否   |                 |                                           |
| local  | Local  | 否   |                 |                                           |

`remote` 或 `local` 之中必须选一个。

### Remote

| 名称   | 类型   | 必选 | 校验规则         | 说明                                        |
|--------|--------|------|------------------|-------------------------------------------|
| url    | string | 是   | must be valid URI | 指向 OPA 服务的 url，如 `http://127.0.0.1:8181/` |
| policy | string | 是   | min_len: 1       | OPA 策略的名称                                 |
| timeout | [Duration](../type.md#duration) | 否    |            | http 客户端超时时间                              |

### Local

| 名称   | 类型   | 必选 | 校验规则 | 说明         |
|--------|--------|------|----------|--------------|
| text   | string | 是   | min_len: 1 | 策略代码     |

## 数据交换

假设原始的客户端请求为：

```shell
GET /?a=1&b= HTTP/1.1
Host: localhost:10000
Pet: dog
Fruit: apple
Fruit: banana
```

这是 HTNN 发送给 OPA 的 JSON 数据：

```json
{
    "input": {
        "request": {
            "scheme": "http",
            "path": "/",
            "query": {
                "a": "1",
                "b": ""
            },
            "method": "GET",
            "host": "localhost:10000",
            "headers": {
                "fruit": "apple,banana",
                "pet": "dog"
            }
        }
    }
}
```

注意：

* `method` 总是大写，而 `host`、`headers` 和 `scheme` 总是小写。
* 如果客户端发送的 `:authority` 头包含端口，则 `host` 将包含该端口。
* 同名的多个 `headers` 和 `query` 将用 ',' 连接。

数据可以在 OPA 中作为 `input` 文档读取。无论是本地模式还是远程模式都用同样的数据。

OPA 策略应该定义一个布尔值 `allow` 并使用它来指示请求是否被允许。

这是 OPA 发回给 HTNN 的 JSON 数据，由配置的策略设置：

```json
{
    "result": {
        "allow": true
    }
}
```

* `allow` 表示请求是否被允许。

## 用法

### 与远程 OPA 服务交互

首先，假设我们有正在运行中的，名为 `opa.service` 的 Open Policy Agent。

让我们添加一个策略：

```shell
curl -X PUT 'opa.service:8181/v1/policies/test' \
    -H 'Content-Type: text/plain' \
    -d 'package test
import input.request
default allow = false
# 只允许 GET 请求
allow {
    request.method == "GET"
}'
```

假设我们有以下附加到 `localhost:10000` 的 HTTPRoute，后端服务器监听端口 `8080`：

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
    opa:
      config:
        remote:
          url: "http://opa.service:8181"
          policy: test
```

如您所见，策略 `test` 将被用来评估我们发送给 OPA 服务的输入数据。

现在，来进行测试：

```shell
curl -i -X GET localhost:10000/echo
HTTP/1.1 200 OK
```

如果我们尝试使用其他方法发起请求，请求将会失败：

```shell
curl -i -X POST localhost:10000/echo -d "AA"
HTTP/1.1 403 Forbidden
```

### 与本地策略规则交互

我们也可以直接配置策略规则。假设我们为 `http://localhost:10000/echo` 提供了如下配置：

```yaml
opa:
  config:
    local:
      text: |
        package test
        import input.request
        default allow = false
        # 仅允许 GET 请求
        allow {
            request.method == "GET"
        }
```

现在，来进行测试：

```shell
curl -i -X GET localhost:10000/echo
HTTP/1.1 200 OK
```

如果我们尝试使用其他方法发起请求，请求将会失败：

```shell
curl -i -X POST localhost:10000/echo -d "AA"
HTTP/1.1 403 Forbidden
```
