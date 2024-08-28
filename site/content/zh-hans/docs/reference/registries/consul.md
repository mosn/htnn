---
title: Consul
---

## 说明

`consul` registry 对接 [Consul](https://developer.hashicorp.com/consul) 服务发现，将服务信息转换成 `ServiceEntry`。

## 配置

| 名称                     | 类型                       | 必选 | 校验规则                 | 说明                 |
|------------------------|--------------------------|------|----------------------|--------------------|
| serverUrl              | string                   | 是   | must be valid URI    | Consul URL         |
| namespace              | string                   | 否   |                      | Consul namespace   |
| dataCenter             | string                   | 否   |                      | Consul datacenter  |
| token                  | string                   | 否   |                      | Consul token       |
| serviceRefreshInterval | [Duration](../type.md#duration) | 否   | gte: 1s              | 轮询服务列表的间隔。默认为 30s。 |

## 用法

假设我们的 Consul 运行在 `172.0.0.1:8500`，则可以通过以下配置对接它：

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

对于一个 tag 为`tag1`，serviceName 为`service1`，namespace 为 `public`，datacenter 为 `dc1`，名称为 `svr`，metadata 为 `{"type":"server"}`，IP 为 `192.168.0.1`，port 为 8080 的注册服务，将生成如下配置：

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

`hosts` 和 `ServiceEntry` 的 `name` 是一致的，格式为 `$tag_name.$consul_namespace.$consul_datacenter.$service_registry_name.consul`。`_` 会被转换成 `-`，大写字母会变小写。如果 host 中有些配置为空，则会自动省略该配置。

生成的配置中，`protocol` 为 HTTP。如果是其他协议，可以在注册信息的 metadata 的 `protocol` 字段指定协议名称。目前支持的协议如下（不区分大小写）：

- http
- https
- grpc
- http2
- mongo
- tcp
- tls

在 HTTPRoute 中，我们可以在 `backendRefs` 引用生成的配置：

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