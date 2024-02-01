---
title: Nacos
---

## 说明

`nacos` registry 对接 [Nacos](https://nacos.io/) 服务发现，将服务信息转换成 `ServiceEntry`。该 registry 支持的是 V1 API，但根据 Nacos OpenAPI 文档，亦可用于对接 Nacos 2.x。

> Nacos 2.X 版本兼容 Nacos1.X 版本的OpenAPI, 请参考文档Nacos1.X OpenAPI使用。
>
> https://nacos.io/zh-cn/docs/v2/guide/user/open-api.html

## 配置

| 名称                     | 类型                            | 必选 | 校验规则          | 说明                                         |
|--------------------------|---------------------------------|------|-------------------|----------------------------------------------|
| serverUrl                | string                          | 是   | must be valid URI | Nacos URL                                    |
| namespace                | string                          | 否   |                   | Nacos namespace。默认为 "public"。           |
| groups                   | string[]                        | 否   | min_len = 1       | Nacos group 列表。默认为 ["DEFAULT_GROUP"]。 |
| service_refresh_interval | [Duration](../../type#duration) | 否   | gte: 1s           | 轮询服务列表的间隔。默认为 30s。             |

Nacos 1.x 没有提供订阅当前服务列表的接口，所以只能通过轮询来获取服务列表。配置一个较小的值可以更快得知服务被删除，但是会给 Nacos 带来更大的压力。

注意：由于心跳间隔、网络延迟等原因，服务的变化可能需要几十秒之后才会引起 `ServiceEntry` 改变。尤其是因为 https://github.com/nacos-group/nacos-sdk-go/issues/139，服务中最后一个示例的移除不会导致 `ServiceEntry` 改变。另外，为了避免因为轮询失败或 Nacos 暂时不可用导致 `ServiceEntry` 被错误删除，只有在 registry 配置变化时，才会清除生成的 `ServiceEntry`。

## 用法

假设我们的 Nacos 运行在 `172.0.0.1:8848`，则可以通过以下配置对接它：

```yaml
apiVersion: mosn.io/v1
kind: ServiceRegistry
metadata:
  name: default
spec:
  type: nacos
  config:
    serverUrl: http://172.0.0.1:8848
```

对于一个 namespace 为 `public`，group 为 `prod`，名称为 `svr`，metadata 为 `{"type":"server"}`，IP 为 `192.168.0.1`，port 为 8080 的注册服务，将生成如下配置：

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

`hosts` 和 `ServiceEntry` 的 `name` 是一致的，格式为 `$service_name.$nacos_group.$nacos_namespace.$service_registry_name.nacos`。`_` 会被转换成 `-`，大写字母会变小写。

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