---
title: Listener Patch
---

## 说明

`listenerPatch` 插件允许用户直接给 Gateway 对应的 Envoy Listener 资源打补丁。

## 属性

|       |          |
|-------|----------|
| Type  | General  |
| Order | Listener |

## 配置

请查阅对应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.29.5/api-v3/config/listener/v3/listener.proto#envoy-v3-api-msg-config-listener-v3-listener)。

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

下面的配置将一个基于文件的 access logger 添加至该网关：

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
    accessLog:
    - name: envoy.access_loggers.file
      typedConfig:
        "@type": type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog
        path: /dev/stdout
        logFormat:
          textFormatSource:
            inlineString: "create listener access log: %DOWNSTREAM_LOCAL_ADDRESS% %EMIT_TIME%\n"
```
