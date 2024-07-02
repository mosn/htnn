---
title: TLS Inspector
---

## Description

The `tlsInspector` plugin adds a TLS Inspector listener filter to the targeted Gateway.

## Attribute

|       |          |
|-------|----------|
| Type  | General  |
| Order | Listener |

## Configuration

See the corresponding [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/listeners/listener_filters/tls_inspector).

## Usage

Assumed we have the Gateway below listening to `localhost:10000`:

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

The configuration below adds TLS Inspector listener filter to it:

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
