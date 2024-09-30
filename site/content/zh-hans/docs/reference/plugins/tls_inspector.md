---
title: TLS Inspector
---

## 说明

`tlsInspector` 插件为目标 Gateway 添加了 TLS 检查器监听过滤器。

## 属性

|        |              |
|--------|--------------|
| Type   | General      |
| Order  | Listener     |
| Status | Experimental |

## 配置

请查阅对应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/listeners/listener_filters/tls_inspector)。

## 用法

假设我们有以下的 Gateway 在 `localhost:10000` 上监听：

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: default
spec:
  gatewayClassName: istio
  listeners:
  - name: default
    hostname: "*"
    port: 10000
    protocol: HTTP
```

下面的配置将 TLS 检查器监听过滤器添加至该网关：

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: default
  filters:
    tlsInspector:
      config: {}
```
