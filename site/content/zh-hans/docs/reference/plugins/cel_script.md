---
title: CEL Script
---

## 说明

`celScript` 插件通过执行用户配置的 [CEL 表达式](../../expr) 来判断当前请求能否访问上游。相对于静态的 Go 代码，CEL 表达式允许运行时动态配置；相对于复杂的规则文件，CEL 表达式执行起来更快；相对于 Lua 脚本，CEL 表达式在沙盒环境下运行，更加安全。

## 属性

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Traffic |

## 配置

| 名称    | 类型   | 必选 | 校验规则 | 说明                                                                     |
|---------|--------|------|----------|--------------------------------------------------------------------------|
| allowIf | string | 否   |          | 判断能否访问的表达式。如果表达式执行结果为 false，则返回 403 HTTP 状态码 |

## 用法

假设我们提供了如下配置到 `http://localhost:10000/`：

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
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend
      port: 8080
---
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    celScript:
      config:
        allowIf: 'request.path() == "/echo" && request.method() == "GET"'
```

`allowIf` 表达式要求请求路径为 `/echo`，方法为 `GET`。

对 `/echo` 路径发起 GET 请求，会成功：

```
$ curl http://localhost:10000/echo
HTTP/1.1 200 OK
```

对 `/echo` 路径发起 POST 请求，会失败：

```
$ curl -X POST http://localhost:10000/echo
HTTP/1.1 403 Forbidden
```
