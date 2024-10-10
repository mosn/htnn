---
title: Plugin development
---

## How to write a plugin

There are two types of HTNN plugins: Native plugins, which are converted to Envoy's Filter configuration at runtime, and Go plugins, which run in the Go runtime embedded in the Envoy. Unless otherwise noted, plugins in the following text refer to Go plugins.

Assume you are at the root of this project.

1. Create a directory under `./types/plugins/`. The directory name must be in go package style, like `keyauth`.
2. Think about the configuration and write down it into `./types/plugins/$yourplugin/config.proto`. Then run `make gen-proto`. The `proto` file uses [proto-gen-valdate](https://github.com/bufbuild/protoc-gen-validate?tab=readme-ov-file#constraint-rules) to define validation. The plugin name must be in camel style, like `keyAuth`. The configuration fields must be in snake style, like `connect_timeout`. The enum value must be in upper snake style, like `HEADER`. See the [official protobuf style](https://protobuf.dev/programming-guides/style/) for the details.
3. Refer to plugins of the same type and decide on the type and order of your plugin.
4. Add your plugin's package into `./types/plugins/plugins.go`.
5. Create a directory under `./plugins/plugins/`, with the same name created in step one. Finish the plugin. Don't forget to write tests. If your plugin is simple, you can write integration test only. You can take `./plugins/plugins/demo` as an example. The doc of the API used in the plugin is in their comments.
6. Add the doc of the plugin in the `site/content/$your_language/docs/reference/plugins/$your_plugin.md`. You can choose to write doc under Simplified Chinese or English, depending on which is your prime language. We have [tool](https://github.com/mosn/htnn/tree/main/site#cmdtranslator) to translate it to other languages.
7. Add your plugin's package into `./plugins/plugins.go`. Go to `./plugins`, then run `make build-so`. Now the plugin is compiled into `libgolang.so` under the current directory.
8. Add integration test in the `./plugins/tests/integration/`. For how to run the integration test, please read [Plugin Integration Test Framework](./plugin_integration_test_framework.md).

You can also write the plugin outside HTNN project, please see [the guide to modify HTNN](./get_involved.md).

### Plugin types

Each plugin should belong to one type. You can specify the plugin's type in its `Type` method. Here are the types:

* `Security`: Plugins like WAF, request validation, etc.
* `Authn`: Plugins do authentication
* `Authz`: Plugins do authorization
* `Traffic`: Plugins do traffic control
* `Transform`: Plugins do request/response transform
* `Observability`: Plugins do observability
* `General`: Else plugins

### Plugin order

We define a fixed order for each plugin.
The order is combined into two parts: the order group and the operation. The order of plugins is first compared by its group.
Then the order of plugins in the group is decided by the operation.
For plugins which have the same operation, they are sorted by alphabetical order.

Here are the order group (sorted from first to last):

The first three order groups are reserved for Native plugins.

* `Listener`: plugins relative to [Envoy listener filters](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/listener_filters/listener_filters).
* `Network`: plugins relative to [Envoy network filters](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/network_filters/network_filters).
* `Outer`: First position for plugins running in HTTP.

Now goes the Go plugins:

* `Access`
* `Authn`
* `Authz`
* `Traffic`
* `Transform`
* `Unspecified`
* `BeforeUpstream`
* `Stats`

End of the Go plugins.

* Istio's extensions go here
* `Inner`: Last position. It's reserved for Native plugins.

There are three kinds of operation: `OrderOperationInsertFirst`, `OrderOperationInsertLast` and `OrderOperationNop`. Each kind means `First`, `Last` and `Middle`.

You can specify the plugin's type in its `Order` method.
If a plugin doesn't claim its order, it will be put into `OrderPositionUnspecified` group, with the operation `OrderOperationNop`.

If you want to configure a plugin in different positions, you can define the plugin as the base class,
and register its derived classes. Please check [this](https://github.com/mosn/htnn/blob/main/api/pkg/plugins/plugins_test.go) for the example.

## Filter manager

The HTNN project introduces filter manager between the Envoy Go filter and the Go Plugins.

Filter manager makes the features below possible:

* Go plugins can be run in goroutine by default, ensure the business logic is non-blocking.
* Reduce CGO calls and increase Go side cache hit.
* Allow additional workflow which is different from Envoy, for example, running extra plugins according to the authenticated user.

### Design of the filter manager

Assumed we have three plugins called `A`, `B` and `C`.
For each plugin, the calling order of callbacks is:

1. DecodeHeaders
2. DecodeData (if request body exists)
3. DecodeTrailers (if request trailers exists)
4. EncodeHeaders
5. EncodeData (if response body exists)
6. EncodeTrailers (if response trailers exists)
7. OnLog

Between plugins, the order of invocation is determined by the order of the plugins. Suppose plugin `A` is in the `Authn` group, `B` is in `Authz`, and `C` is in `Traffic`.

When processing the request (Decode path), the calling order is `A -> B -> C`.
When processing the response (Encode path), the calling order is `C -> B -> A`.
When logging the request (OnLog), the calling order is `A -> B -> C`.

By using the plugin order instead of plugin name, we can also say:

When processing a request, the call order is `Authn -> Authz -> Traffic`.
When processing a response, the call order is `Traffic -> Authz -> Authn`.
When logging requests, the call order is `Authn -> Authz -> Traffic`.

![filter manager](/images/filtermanager_main_path.jpg)

Note that this picture shows the main path. The execution path may have slight differences. For example,

* If the request doesn't have body, the `DecodeData` won't be called.
* If the request contains trailers, the `DecodeTrailers` will be called after the body is handled.
* If the request is replied by Envoy before being sent to the upstream, we will leave the Decode path and enter the Encode path.
For example, if the plugin B rejects the request with some custom headers, the Decode path is `A -> B` and the Encode path is `C -> B -> A`.
The custom headers will be rewritten by the plugins. This behavior is equal to Envoy.

In some situations, we need to stop the iteration of header filter, then read the whole body. For instance,

1. Authorization with request body.
2. Modify the body, and change the headers (`content-length` and so on).

Therefore, we introduce a group of new types:

* `WaitAllData`: a `ResultAction` returns from the `DecodeHeaders` or `EncodeHeaders`
* `DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction`
* `EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction`

`WaitAllData` can be used to decide if the body needs to be buffered, according to the configuration and the headers.

If `WaitAllData` is returned from `DecodeHeaders`, we will:

1. buffer the whole body
2. execute the `DecodeData` and `DecodeTrailers` of previous plugins
3. execute the `DecodeRequest` of this plugin
4. back to the original path, continue to execute the `DecodeHeaders` of the next plugin

![filter manager, with DecodeWholeRequestFilter, buffer the whole request](/images/filtermanager_sub_path.jpg)

Note: `DecodeRequest` is only executed if `DecodeHeaders` returns `WaitAllData`. So if `DecodeRequest` is defined, `DecodeHeaders` must be defined as well. When both `DecodeRequest` and `DecodeData/DecodeTrailers` are defined in the plugin: if `DecodeHeaders` returns `WaitAllData`, only `DecodeRequest` is executed, otherwise, only `DecodeData/DecodeTrailers` is executed.

The same process applies to the Encode path in a reverse order, and the method is slightly different. This time it requires `EncodeHeaders` to return `WaitAllData` to invoke `EncodeResponse`.

Note: `EncodeResponse` is only executed if `EncodeHeaders` returns `WaitAllData`. So if `EncodeResponse` is defined, `EncodeHeaders` must be defined as well. When both `EncodeResponse` and `EncodeData/EncodeTrailers` are defined in the plugin: if `EncodeHeaders` returns `WaitAllData`, only `EncodeResponse` is executed, otherwise, only `EncodeData/EncodeTrailers` is executed.

Currently, `DecodeRequest` is not supported by plugins whose order is `Access` or `Authn`.

## Consumer Plugins

Consumer plugins are a special type of Go plugin. They locate and set a [consumer](../concept/consumer.md) based on the content of the request headers.

A consumer plugin needs to meet the following conditions:

* Both `Type` and `Order` are `Authn`.
* Implements the [ConsumerPlugin](https://pkg.go.dev/mosn.io/htnn/pkg/plugins#ConsumerPlugin) interface.
* Defines the `DecodeHeaders` method, and in this method, it calls `LookupConsumer` and `SetConsumer` to complete the setting of the consumer.

You can take the `keyAuth` plugin as an example to write your own consumer plugin.
