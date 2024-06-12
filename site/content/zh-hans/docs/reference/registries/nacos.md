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
| serviceRefreshInterval   | [Duration](../../type#duration) | 否   | gte: 1s           | 轮询服务列表的间隔。默认为 30s。             |

Nacos 1.x 没有提供订阅当前服务列表的接口，所以只能通过轮询来获取服务列表。配置一个较小的值可以更快得知服务被删除，但是会给 Nacos 带来更大的压力。

如果在 `serverUrl` 里面使用域名，它必须是 FQDN，如 `svc.cluster.local`，而不是 `svc`。

注意：由于心跳间隔、网络延迟等原因，服务的变化可能需要几十秒之后才会引起 `ServiceEntry` 改变。尤其是因为 https://github.com/nacos-group/nacos-sdk-go/issues/139，服务中最后一个示例的移除不会导致 `ServiceEntry` 改变。另外，为了避免因为轮询失败或 Nacos 暂时不可用导致 `ServiceEntry` 被错误删除，只有在 registry 配置变化时，才会清除生成的 `ServiceEntry`。

注意：因为 [nacos-sdk-go](https://github.com/nacos-group/nacos-sdk-go/) 会向文件系统写入日志和缓存，而默认情况下 HTNN 的控制面是以只读模式挂载的，所以会导致无法对接 Nacos 。解决方法是在部署 HTNN 时往 `/log` 和 `/cache` 挂载可写的目录。以通过 helm 安装 HTNN 为例，可以通过以下方式挂载可写的目录：

```shell
helm install htnn-controller htnn/htnn-controller ... -f custom-values.yaml
```

其中 `custom-values.yaml` 包含如下内容：

```yaml
pilot:
  volumes:
  - emptyDir:
      medium: Memory
      # log 被配置成保留 10 个 1M 的日志文件，所以 20M 的空间足够了
      sizeLimit: 20Mi
    name: nacos-log
  - emptyDir:
      medium: Memory
      # 取决于服务发现的数据量
      sizeLimit: 20Mi
    name: nacos-cache
  volumeMounts:
  - name: nacos-log
    mountPath: /log
  - name: nacos-cache
    mountPath: /cache
```

更理想的解决方法是从源头上就不要让 nacos-sdk-go 往本地文件系统里写日志和缓存。毕竟在云原生场景下，将日志和缓存落到本地盘的意义不大。如果你找到让 nacos-sdk-go 不写入文件系统的方法，欢迎更新本文档。

## 用法

假设我们的 Nacos 运行在 `172.0.0.1:8848`，则可以通过以下配置对接它：

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