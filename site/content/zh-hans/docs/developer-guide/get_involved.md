---
title: 如何二次开发 HTNN
---

HTNN 功能代码主要位于下面的模块：

* api/：最基础的模块。它被其他模块引用，并对外提供开发 HTNN 插件所需的接口。
* types/：控制面、数据面和 console 都会依赖该模块提供插件元数据、CRD 定义等内部通用的数据。
* controller/：控制面
* plugins/：数据面（plugin hub）

对于二次开发的用户，一般都只需参考 `controller/` 和 `plugins/` 里面的实现。

## controller

HTNN 在 `controller/` 模块主要做了下面的事：

* 调和 HTNN 的 CRD
* 提供 Native Plugin 框架，允许以插件的方式修改发送给数据面的 xDS
* 提供 Service Registry 机制，允许以 registry 的方式对接外部服务发现系统，转换成数据面内的上游信息

其中和开发者紧密相关的是 Plugin 和 Registry。建议阅读 `plugins/` 和 `registries/` 两个目录下的代码，看看如何使用 HTNN 提供的接口开发功能。

传统上想要修改 istio 的行为有两种方法，

1. 将自己的资源翻译成 EnvoyFilter 写入到 k8s 里，让 istio 处理 EnvoyFilter。
2. 让自己成为 MCP server。调和自己的资源，再把结果通过 MCP 协议给 istio。

HTNN 没有用上述方法，而是通过修改 istio，把自己的流程嵌入到 istio 当中。具体的改动可参考 `patch/istio` 目录下的 patch 文件。之所以不用方法 1，是因为写入带状态的 EnvoyFilter 会导致可观测性和高可用变得更加困难，容易出问题。不用方法 2 的原因在于要想保证策略和路由的一致性，需要将路由也接入到 MCP server 中，否则会出现策略还在调和过程中，路由已经发布上线的情况。将路由也一并在 MCP server 里处理，无疑会是很大的工作量。

如果你在第三方仓库下开发了自己的 Native Plugin 或 Service Registry，想要在 HTNN 中使用，可以通过 patch 的方式将该仓库引入到 istio 当中来。具体参考 https://github.com/mosn/htnn/blob/main/patch/istio/1.21/20240410-htnn-go-mod.patch 这个 patch。注意 import 该仓库时需要把它放到 HTNN 官方插件 package 的后面，这样才能避免被同名的插件覆盖掉。

如果要想运行 HTNN 的控制面，可以参考 `e2e/` 目录下 `make e2e-prepare-controller-image` 的实现，看看如何将嵌入了 HTNN 的 istio 打包成镜像。

## plugins

HTNN 在 `plugins/` 模块下放置在数据面上运行的 Go Plugin 以及它们共享的 pkg 库。

如果你想要在第三方仓库编写 Go Plugin，可以参考 https://github.com/mosn/htnn/tree/main/examples/dev_your_plugin 这个范例。Go Plugin 在数据面上是编译成 shared library 来部署到。当你想要集成 Go Plugin 到数据面时，https://github.com/mosn/htnn/blob/main/examples/dev_your_plugin/cmd/libgolang/main.go 这个文件可以作为模版。

在开发自己的插件之前建议读一下现有插件的代码，尤其是和你开发的功能相似的插件，至少应该看看 [demo](https://github.com/mosn/htnn/tree/main/plugins/plugins/demo) 插件。插件代码位于 `plugins/` 目录下面。关于插件开发的更多信息，请参考[插件开发](./plugin_development.md)文档。

HTNN 提供了一个[插件集成测试框架](./plugin_integration_test_framework.md)，允许在只运行数据面的情况下测试 Go Plugin 的逻辑。在 `dev_your_plugin` 这个范例里也展示了如何在第三方仓库中运行集成测试框架。

每个插件都包含两类对象：`config` 和 `filter`。其中 `config` 负责配置管理，`filter` 负责执行请求级别的逻辑。

### config

在插件开发时，和配置紧密相关的功能都应当放到 `config` 里。

`config` 的粒度可以是 per-route 的，也可以是 per-gateway 或者 per-consumer，取决于插件配置的范围。比如配置 Gateway 级别的 `limitCountRedis`:

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
  namespace: istio-system
spec:
  targetRef:
    group: networking.istio.io
    kind: Gateway
    name: default
  filters:
    limitCountRedis:
      config:
        address: "redis:6379"
        rules:
        - count: 1
          timeWindow: "60s"
```

这时候会创建一个 `limitCountRedis.config` 对象，且这个对象在整个 Gateway `default` 下各个路由里是共享的。

`config` 的生命周期取决于插件配置本身和它所作用的对象。举个例子，如果我们在路由 `vs` 上配置了 `limitCountRedis`:

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
  namespace: istio-system
spec:
  targetRef:
    group: networking.istio.io
    kind: VirtualService
    name: vs
  filters:
    limitCountRedis:
      config:
        address: "redis:6379"
        rules:
        - count: 1
          timeWindow: "60s"
```

在两种情况下 `limitCountRedis.config` 对象会被重新创建：
1. 任何 `vs` 所在的 Gateway 下的路由发生变化
2. 任何指向上一种情况里面的路由的 FilterPolicy 的 spec 发生变化

这是因为上述情况下会创建新的 [RouteConfiguration](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route.proto#envoy-v3-api-msg-config-route-v3-routeconfiguration)，导致 `limitCountRedis.config` 对象被重建。

（TODO：支持增量更新路由，这样只有 `vs` 自己的变化才会引起对象被重建）

由于 `config` 在不同的 Envoy worker 线程的多个请求间是共享的，所以对 `config` 的读写操作必须加锁。

### filter

在插件开发时，和每请求相关的功能都应当放到 `filter` 里。每个请求都会给每个执行的插件创建一个 `filter` 对象。

`filter` 里主要定义下面的方法：

1. DecodeHeaders
2. DecodeData
3. EncodeHeaders
4. EncodeData
5. OnLog

正常情况下，会从上到下执行上述方法。但存在以下特例：

1. 没有 body 则不会执行对应的 DecodeData 和 EncodeData 方法。
2. 由于客户端中断请求时会触发 OnLog 操作，所以当客户端提前中断请求时，OnLog 方法可能和其他方法同时执行。
3. 在 bidirectional stream 之类的请求里，有可能出现同时处理请求体和上游响应的情况，所以 DecodeData 可能和 EncodeHeaders 或 EncodeData 同时执行。

所以当读写 `filter` 时可能遇到并发访问的风险时需要考虑加锁。
