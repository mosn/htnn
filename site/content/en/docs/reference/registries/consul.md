---
title: Consul
---

## Description

The `consul` registry connects to the [Consul](https://developer.hashicorp.com/consul) service discovery and converts service information into `ServiceEntry`.

## Configuration

| Name                   | Type                        | Required | Validation        | Description        |
|------------------------|-----------------------------|----------|-------------------|---------------------|
| serverUrl              | string                      | True     | must be valid URI | Consul URL          |
| namespace              | string                      | False    |                   | Consul namespace    |
| dataCenter             | string                      | False    |                   | Consul datacenter   |
| token                  | string                      | False    |                   | Consul token        |
| serviceRefreshInterval | [Duration](../type.md#duration) | False    | gte: 1s           | Interval for polling the service list. Default is 30s. |

## Usage

Assume our Consul is running at `172.0.0.1:8500`, you can connect to it with the following configuration:

```yaml
apiVersion: htnn.mosn.io/v1
kind: ServiceRegistry
metadata:
  name: default
spec:
  type: consul
  config:
    serverUrl: http://172.0.0.1:8500
```

For a registered service with the tag `tag1`, serviceName `service1`, namespace `public`, datacenter `dc1`, name `svr`, metadata `{"type":"server"}`, IP `192.168.0.1`, and port 8080, the generated configuration would be as follows:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: tag1.service1.public.dc1.svr.consul
spec:
  endpoints:
  - address: 192.168.0.1
    labels:
      type: server
    ports:
      HTTP: 8080
  hosts:
  - tag1.service1.public.dc1.svr.consul
  location: MESH_INTERNAL
  ports:
  - name: HTTP
    number: 8080
    protocol: HTTP
  resolution: STATIC
```

The `hosts` and the `ServiceEntry` `name` are consistent, with the format `$tag_name.$consul_namespace.$consul_datacenter.$service_registry_name.consul`. Underscores (`_`) will be converted to hyphens (`-`), and uppercase letters will be converted to lowercase. If some configurations in the host are empty, they will be automatically omitted.

In the generated configuration, the `protocol` is HTTP. If it's another protocol, you can specify the protocol name in the `protocol` field of the metadata in the registration information. The currently supported protocols are as follows (case-insensitive):

- http
- https
- grpc
- http2
- mongo
- tcp
- tls

In the HTTPRoute, we can reference the generated configuration in `backendRefs`:

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
    - name: tag1.service1.public.dc1.svr.consul
      port: 8080
      group: networking.istio.io
      kind: Hostname
```
