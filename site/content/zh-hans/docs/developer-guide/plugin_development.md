---
title: 插件开发
---

## 如何编写插件

HTNN 的插件分成两种：Go 插件和 Native 插件。Native 插件在运行时会被转换成 Envoy 的 Filter 配置。Go 插件则是运行在嵌入 Envoy 的 Go 运行时当中。如无特殊说明，下文的插件均指 Go 插件。

假设您位于此项目的根目录。

1. 在 `./types/plugins/` 下创建一个目录。目录名必须使用蛇形命名法，如 `keyauth`。
2. 思考配置并将其写入 `./plugins/$yourplugin/config.proto`。然后运行 `make gen-proto`。`proto` 文件使用 [proto-gen-valdate](https://github.com/bufbuild/protoc-gen-validate?tab=readme-ov-file#constraint-rules) 定义验证。插件名必须使用驼峰式命名法，如 `keyAuth`。配置字段必须使用蛇形命名法，如 `connect_timeout`。枚举值必须使用大写蛇形命名法，如 `HEADER`。详细信息请参见[官方 protobuf 风格指南](https://protobuf.dev/programming-guides/style/)。
3. 参考同类型的插件，决定你的插件类型和顺序。
4. 在 `./types/plugins/plugins.go` 中引入你的插件。
5. 在 `./plugins/plugins/` 下创建一个目录，文件名必须和步骤一里面的名字一致。完成插件编写。不要忘记编写测试。如果您的插件简单，您可以仅编写集成测试。您可以以 `./plugins/plugins/demo` 为例。插件开发中涉及的 API 的文档位于这些 API 的注释当中。
6. 在 `site/content/$your_language/docs/reference/plugins/$your_plugin.md` 中添加插件文档。您可以选择使用简体中文或英文编写文档，这取决于哪种是您擅长的语言。我们有 [工具](https://github.com/mosn/htnn/tree/main/site#cmdtranslator) 可以将其翻译成其他语言。
7. 在 `./plugins/plugins.go` 中引入你的插件。前往目录 `./plugins`，运行 `make build-so`。插件将被编译入当前目录下的 `libgolang.so`。
8. 在 `./plugins/tests/integration/` 中添加集成测试。有关如何运行集成测试，请阅读[插件集成测试框架](./plugin_integration_test_framework.md)。

您还可以在 HTNN 项目之外编写插件，请参见 [二次开发 HTNN 指南](./get_involved.md)。

### 插件类型

每个插件应属于一种类型。您可以在其 `Type` 方法中指定插件的类型。这里有几种类型：

* `Security`：像 WAF、请求验证等插件。
* `Authn`：执行认证的插件
* `Authz`：执行授权的插件
* `Traffic`：执行流量控制的插件
* `Transform`：执行请求/响应转换的插件
* `Observability`：执行可观察性的插件
* `General`：其他插件

### 插件顺序

我们为每个插件定义了固定顺序。
顺序由两部分组合而成：顺序组（Order Group）和 操作（Operation）。插件的顺序首先通过其组进行比较。
然后，同组内插件的顺序由操作决定。
对于操作相同的插件，它们按字母顺序排序。
以下是顺序组（从第一个到最后一个排序）：

前三个顺序组为 Native 插件保留：

* `Listener`: 和 [Envoy listener filters](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/listener_filters/listener_filters) 相关的插件。
* `Network`: 和 [Envoy network filters](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/network_filters/network_filters) 相关的插件。
* `Outer`：在 HTTP 上运行最前端的插件。

现在开始是 Go 插件：

* `Access`
* `Authn`
* `Authz`
* `Traffic`
* `Transform`
* `Unspecified`
* `BeforeUpstream`
* `Stats`

Go 插件结束。

* Istio 的扩展在这里
* `Inner`：最后位置。它为 Native 插件保留。

有三种操作类型：`OrderOperationInsertFirst`、`OrderOperationInsertLast` 和 `OrderOperationNop`。他们分别意味着 `First`、`Last` 和 `Middle`。

您可以在其 `Order` 方法中指定插件的类型。
如果插件没有声明其顺序，它将被放入 `OrderPositionUnspecified` 组，操作为 `OrderOperationNop`。

如果您想在不同位置配置插件，您可以将插件定义为基类，
并注册其派生类。请检查[此示例](https://github.com/mosn/htnn/blob/main/api/pkg/plugins/plugins_test.go)。

## Filter manager

HTNN 项目在 Envoy Go Filter 和 Go 插件之间引入了 filter manager。

filter manager 实现了以下特性：

* Go 插件可以默认在协程中运行，确保业务逻辑非阻塞。
* 减少 CGO 调用并提高 Go 端缓存命中率。
* 允许与 Envoy 不同的执行流程，例如，根据认证用户运行额外插件。

### Filter manager 设计

假设我们有三个称为 `A`、`B` 和 `C` 的插件。
对于每个插件，回调的调用顺序是：

1. DecodeHeaders
2. DecodeData（如果存在 request body）
3. DecodeTrailers（如果存在 request trailer）
4. EncodeHeaders
5. EncodeData（如果存在 response body）
6. EncodeTrailers（如果存在 response trailer）
7. OnLog

在插件之间，调用顺序由插件顺序决定。假设 `A` 插件在 `Authn` 组，`B` 在 `Authz`，`C` 在 `Traffic`。
处理请求时（Decode 路径），调用顺序是 `A -> B -> C`。
处理响应时（Encode 路径），调用顺序是 `C -> B -> A`。
记录请求时（OnLog），调用顺序是 `A -> B -> C`。

如果我们使用插件顺序而不是插件名称，则调用顺序可以被描述为：
处理请求时，调用顺序是 `Authn -> Authz -> Traffic`。
处理响应时，调用顺序是 `Traffic -> Authz -> Authn`。
记录请求时，调用顺序是 `Authn -> Authz -> Traffic`。

![过滤器管理器](/images/filtermanager_main_path.jpg)

请注意，这张图片显示的是主路径。实际执行路径可能有细微差别。例如，

* 如果请求没有 body，将不会调用 `DecodeData`。
* 如果请求中有 trailers，则会在处理完 body 后调用 `DecodeTrailers` 处理 trailers。
* 如果 Envoy 在发送给上游之前回复了请求，我们将离开 Decode 路径并进入 Encode 路径。例如，如果插件 B 用一些自定义头拒绝了请求，Decode 路径是 `A -> B`，Encode 路径是 `C -> B -> A`。自定义头将被该路径上的插件重写。这种行为和 Envoy 的处理方式一致。

在某些情况下，我们需要中止 header filter 的执行，直到收到整个 body。例如，

1. 鉴权操作时需要检查请求体
2. 修改 body，并更新某些 headers（`content-length` 等）

因此，我们引入了一组新类型：

* `WaitAllData`: 由 `DecodeHeaders` 或 `EncodeHeaders` 返回的 `ResultAction`
* `DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction`
* `EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction`

`WaitAllData` 可用于根据配置和 headers 决定是否需要缓冲 body。

如果 `DecodeHeaders` 返回 `WaitAllData`，我们将：

1. 缓冲整个 body
2. 执行之前插件的 `DecodeData/DecodeTrailers`
3. 执行此插件的 `DecodeRequest`
4. 回到原始路径，继续执行下一个插件的 `DecodeHeaders`

![过滤器管理器，带有 DecodeWholeRequestFilter，缓冲整个请求](/images/filtermanager_sub_path.jpg)

注意：`DecodeRequest` 仅在 `DecodeHeaders` 返回 `WaitAllData` 时才被执行。所以如果定义了 `DecodeRequest`，一定要定义 `DecodeHeaders`。如果插件里同时定义了 `DecodeRequest` 和 `DecodeData/DecodeTrailers`，执行哪一个方法取决于 `DecodeHeaders` 是否返回 `WaitAllData`：如果 `DecodeHeaders` 返回 `WaitAllData`，只有 `DecodeRequest` 会运行，否则只有 `DecodeData/DecodeTrailers` 会运行。

同样的过程适用于方向相反的 Encode 路径，且方式略有不同。此时需要由 `EncodeHeaders` 返回 `WaitAllData`，调用方法 `EncodeResponse`。

注意：`EncodeResponse` 仅在 `EncodeHeaders` 返回 `WaitAllData` 时才被执行。所以如果定义了 `EncodeResponse`，一定要定义 `EncodeHeaders`。当插件里同时定义了 `EncodeResponse` 和 `EncodeData/EncodeTrailers`：如果 `EncodeHeaders` 返回 `WaitAllData`，只有 `EncodeResponse` 会运行，否则只有 `EncodeData/EncodeTrailers` 会运行。

目前如果配置了消费者插件，顺序为 `Access` 或 `Authn` 的插件的 `DecodeRequest` 方法将不会被执行。

## 消费者插件

消费者插件是一种特殊的 Go 插件。它根据请求头中的内容查找并设置[消费者](../concept/consumer.md)。

一个消费者插件需要满足下面的条件：

* `Type` 和 `Order` 都是 `Authn`。
* 实现 [ConsumerPlugin](https://pkg.go.dev/mosn.io/htnn/pkg/plugins#ConsumerPlugin) 接口。
* 定义 `DecodeHeaders` 方法，且在该方法里调用 `LookupConsumer` 和 `SetConsumer` 完成消费者的设置。

您可以以 `keyAuth` 插件为例，编写自己的消费者插件。

## 为什么我的插件没有被执行

首先确保插件已经被加载。Envoy 在加载 Go 插件时会打印如下日志：

```text
[plugins] "msg"="register plugin" "name"="casbin"
```

其次，当 Envoy 收到 Go 插件配置时，且日志等级在 info 或以下时，会打印如下日志：

```text
[2024-10-16 12:02:28.505][1][info][golang] [contrib/golang/common/log/cgo.cc:18] receive consumer configuration: {"auth":{"hmacAuth":"{\"accessKey\":\"ak\",\"secretKey\":\"sk\",\"signedHeaders\":[\"x-custom-a\"],\"algorithm\":\"HMAC_SHA256\"}","keyAuth":"{\"key\":\"rick\"}"}}
...
[2024-10-16 12:02:29.033][1][info][golang] [contrib/golang/common/log/cgo.cc:18] receive filtermanager config: {"namespace":"ns", "plugins":[{"config":{"keys":[{"name":"Authorization", "source":"HEADER"}, {"name":"ak", "source":"QUERY"}]}, "name":"keyAuth"}, {"config":{"deny_if_no_consumer":true}, "name":"consumerRestriction"}]}
```

请检查和预期是否一致。其中 `filtermanager config` 里面的 plugins 里的插件顺序就是插件执行的顺序。如果插件已经加载，且在目标路由上有对应的配置信息，但是插件没有被执行，可能是因为：

* 插件的方法定义不合预期，比如定义了 `DecodeRequest` 方法但 `DecodeHeaders` 没有返回 `WaitAllData`。
* 优先级在该插件之前的插件提前中止了请求，比如前面的认证插件返回 403。
* HTNN 的 bug。

可以通过以下方法查看具体执行的插件和执行顺序：

* 将日志等级降到 debug，我们将看到具体的插件执行日志： `finish running plugin coverage, method: DecodeHeaders`。
* 设置 debugMode 插件，并将 slow threshold 降为 0。这样每个请求都会在应用日志中打印执行过的插件信息。
