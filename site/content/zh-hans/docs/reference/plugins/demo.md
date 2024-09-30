---
title: Demo
---

## 说明

`demo` 插件用于展示如何向 htnn 添加一个插件。

## 属性

|        |              |
|--------|--------------|
| Type   | General      |
| Order  | Unspecified  |
| Status | Experimental |

## 配置

| 名称     | 类型   | 必选 | 校验规则   | 说明                                                                          |
|----------|--------|------|------------|--------------------------------------------------------------------------------|
| hostName | string | 是   | min_len: 1 | 请求头名称，我们将通过这个请求头向上游传递我们的问候 `hello, ...`                                |

## 用法

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

通过应用以下配置，此插件将在请求中插入一个请求头 `John Doe`, 内容为 `hello, $guest_name`。`$guest_name` 的值默认为空。

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
    demo:
      config:
        hostName: "John Doe"
```
