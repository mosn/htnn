---
title: Outer Ext Proc
---

## 说明

`outerExtProc` 插件通过利用 Envoy 的 `ext_proc` 过滤器，允许在请求处理过程中与外部服务通讯，用外部服务的处理结果来改写请求或响应。

因为 Envoy 使用洋葱模型来代理请求，执行顺序是：

1. 请求开始
2. 运行 `Outer` 组的 `outerExtProc` 和其他插件
3. 运行其他插件
4. 代理到上游
5. 运行其他插件处理响应
6. 运行 `Outer` 组的 `outerExtProc` 和其他插件处理响应
7. 请求结束

## 属性

|       |         |
|-------|---------|
| Type  | General |
| Order | Outer   |

## 配置

具体的配置字段参见对应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.29.4/api-v3/extensions/filters/http/ext_proc/v3/ext_proc.proto#envoy-v3-api-msg-extensions-filters-http-ext-proc-v3-extprocoverrides)。

External Processing 的工作原理参见 [Envoy 的 External Processing 机制介绍](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/http/http_filters/ext_proc_filter.html)。

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

我们在 k8s 内运行一个 namespace 为 default，名字为 processor 的服务。它监听 8080 端口，实现了 Envoy 的 External Processing 协议。

通过应用下面的配置，我们可以用 processor 服务的响应修改对 `http://localhost:10000/` 的请求：

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
    outerExtProc:
      config:
        grpcService:
          envoyGrpc:
            clusterName: outbound|8080||processor.default.svc.cluster.local
```

这里通过 clusterName 指定了目标 External Processing 服务的地址，命名格式为 `outbound|$port||$FQDN`。在配置目标服务之前，需要确保 istio 已经将服务地址同步到数据面上。我们可以使用 `istioctl pc cluster $data_plane_id` 来查询可以访问的服务列表。
