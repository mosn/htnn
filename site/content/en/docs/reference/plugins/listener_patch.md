---
title: Listener Patch
---

## Description

The `listenerPatch` plugin allows users to directly patch the Listener resource of Envoy generated from the corresponding Gateway.

## Attribute

|       |          |
|-------|----------|
| Type  | General  |
| Order | Listener |

## Configuration

Please refer to the corresponding [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.29.5/api-v3/config/listener/v3/listener.proto#envoy-v3-api-msg-config-listener-v3-listener).

## Usage

Assume we have the following Gateway listening on `localhost:10000`:

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

The following configuration will add a file-based access logger to the gateway:

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
