---
title: Consumer Restriction
---

## 说明

`consumerRestriction` 插件根据配置，判断当前的消费者是否有访问权限。如果当前不存在消费者或消费者没有访问权限，则返回 403 HTTP 状态码。

## 属性

|       |       |
|-------|-------|
| Type  | Authz |
| Order | Authz |

## 配置

| 名称  | 类型  | 必选 | 校验规则 | 说明               |
|-------|-------|------|----------|--------------------|
| allow | Rules | 否   |          | 允许访问的规则列表 |
| deny  | Rules | 否   |          | 禁止访问的规则列表 |

`allow` 和 `deny` 之间只能配置一个。

### Rules

| 名称  | 类型   | 必选 | 校验规则     | 说明          |
|-------|--------|------|--------------|---------------|
| rules | Rule[] | 是   | min_items: 1 | 规则列表 |

### Rule

| 名称      | 类型       | 必选  | 校验规则   | 说明                        |
|---------|----------|-----|------------|---------------------------|
| name    | string   | 是   | min_len: 1 | Consumer 名称               |
| methods | string[] | 否   | must be uppercase | Consumer 允许/禁止的 HTTP 方法列表 |

## 用法

首先，让我们创建两个消费者：

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

假设我们提供了如下配置到 `http://localhost:10000/time_travel`，并且有一个后端服务器监听端口 `8080`：

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
        type: Exact
        value: /time_travel
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
    consumerRestriction:
      config:
        allow:
          rules:
          - name: doraemon
```

`doraemon` 可以访问 `/time_travel`，除此以外的消费者都无法访问该路由。

让我们试一试：

```shell
$ curl -I http://localhost:10000/time_travel -H "Authorization: doraemon"
HTTP/1.1 200 OK
$ curl -I http://localhost:10000/time_travel -H "Authorization: rick"
HTTP/1.1 403 Forbidden
```

如果想用黑名单，则用 `deny` 替换 `allow`：

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
