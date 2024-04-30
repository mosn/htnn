---
title: Outer Lua
---

## 说明

`outerLua` 插件通过利用 Envoy 的 `lua` 过滤器，允许在请求的开始和结束时运行 Lua 代码片段。

因为 Envoy 使用洋葱模型来代理请求，执行顺序是：

1. 请求开始
2. 运行 `Outer` 组的 outerLua 和其他插件
3. 运行其他插件
4. 代理到上游
5. 运行其他插件处理响应
6. 运行 `Outer` 组的 outerLua 和其他插件处理响应
7. 请求结束

## 属性

|       |         |
|-------|---------|
| Type  | General |
| Order | Outer   |

## 配置

参见对应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.29.2/configuration/http/http_filters/lua_filter)。

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

通过应用下面的配置，我们可以用自定义响应中断对 `http://localhost:10000/` 的请求：

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
    outerLua:
      config:
        sourceCode:
          inlineString: |
            function envoy_on_request(handle)
              local resp_headers = {[":status"] = "200"}
              local data = "hello, world"
              handle:respond(
                resp_headers,
                data
              )
            end
```

我们可以测试一下：

```
$ curl http://localhost:10000/
HTTP/1.1 200 OK
content-length: 12
...
hello, world
```
