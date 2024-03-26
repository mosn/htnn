---
title: Nacos
---

## Description

The `nacos` registry interfaces with [Nacos](https://nacos.io/) service discovery, converting service information into `ServiceEntry`. This registry supports the V1 API, but according to the Nacos OpenAPI documentation, it can also be used to interface with Nacos 2.x.

> Nacos 2.X is compatible with Nacos 1.X OpenAPI, please refer to the document Nacos1.X OpenAPI.
>
> https://nacos.io/en/docs/v2/guide/user/open-api/

## Configuration
| Name                     | Type                            | Required | Validation        | Description                                            |
|--------------------------|---------------------------------|----------|-------------------|--------------------------------------------------------|
| serverUrl                | string                          | True     | must be valid URI | Nacos URL                                              |
| namespace                | string                          | False    |                   | Nacos namespace. Default is "public".                  |
| groups                   | string[]                        | False    | min_len = 1       | List of Nacos groups. Default is ["DEFAULT_GROUP"].    |
| service_refresh_interval | [Duration](../../type#duration) | False    | gte: 1s           | Interval for polling the service list. Default is 30s. |

Nacos 1.x does not provide an API to subscribe to the current service list, so polling is the only way to retrieve the service list. Configuring a smaller value can allow for quicker detection of service deletions, but will place more pressure on Nacos.

Note: Due to heartbeat intervals, network latencies, and other factors, it may take several seconds for changes in services to affect the `ServiceEntry`. In particular, because of https://github.com/nacos-group/nacos-sdk-go/issues/139, the removal of the last instance in a service will not lead to a change in `ServiceEntry`. Additionally, to prevent `ServiceEntry` from being mistakenly deleted due to polling failures or temporary unavailability of Nacos, the generated `ServiceEntry` will only be cleared when there are changes to the registry configuration.

## Usage

Assuming our Nacos is running at `172.0.0.1:8848`, it can be interfaced with the following configuration:

```yaml
apiVersion: htnn.mosn.io/v1
kind: ServiceRegistry
metadata:
  name: default
spec:
  type: nacos
  config:
    serverUrl: http://172.0.0.1:8848
```

For a registered service with a namespace of `public`, group of `prod`, name of `svr`, metadata of `{"type":"server"}`, IP of `192.168.0.1`, and port of 8080, the following configuration will be generated:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: svr.prod.public.default.nacos
spec:
  endpoints:
  - address: 192.168.0.1
    labels:
      type: server
    ports:
      HTTP: 8080
  hosts:
  - svr.prod.public.default.nacos
  location: MESH_INTERNAL
  ports:
  - name: HTTP
    number: 8080
    protocol: HTTP
  resolution: STATIC
```

`hosts` and the `name` of the `ServiceEntry` are consistent, formatted as `$service_name.$nacos_group.$nacos_namespace.$service_registry_name.nacos`. `_` will be converted to `-`, and uppercase letters will be changed to lowercase.

In the generated configuration, `protocol` is HTTP. If it's another protocol, it can be specified in the `protocol` field of the metadata in the registration information. The currently supported protocols are as follows (case-insensitive):

- http
- https
- grpc
- http2
- mongo
- tcp
- tls

In HTTPRoute, we can refer to the generated configuration in `backendRefs`:

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
    - name: svr.prod.public.default.nacos
      port: 8080
      group: networking.istio.io
      kind: Hostname
```