---
title: How to develop HTNN to fit your purpose
---

HTNN's functional code is mainly located in the following modules:

* api/: This is the most basic module. It is referenced by other modules and provides interfaces needed to develop HTNN plugins.
* types/: Modules such as the control plane, data plane, and console all depend on this module to provide plugin metadata, CRD definitions, and other internal common data.
* controller/: Control plane
* plugins/: Data plane (plugin hub)

For users who want to develop their functionalities, it is generally only necessary to refer to the implementations in `controller/` and `plugins/`.

## Controller

HTNN in the `controller/` module mainly does the following:

* Reconcile HTNN's CRD
* Provide a Native Plugin framework to allow modifications to the xDS sent to the data plane through plugins.
* Provide a Service Registry mechanism to integrate the external service discovery systems in the form of registries, converting them into upstream information within the data plane.

Plugin and Registry are closely related to developers. It is advisable to read the code in the `plugins/` and `registries/` directories to see how to develop functionalities using HTNN's interfaces.

Traditionally, there are two ways to alter the behavior of Istio:

1. Translate your resources into EnvoyFilter and write them into k8s to let Istio process the EnvoyFilter.
2. Become an MCP server yourself. Reconcile your own resources, then deliver the results to Istio through the MCP protocol.

HTNN does not use the methods above but instead modifies Istio to embed its own process into Istio. The specific changes can be referred to in the `patch/istio` directory's patch files. The reason for not using method 1 is that writing stateful EnvoyFilter leads to observability and high availability becoming more challenging and prone to problems. Method 2 is not used because to ensure the consistency of network policy and network routing, the network routing part also needs to be integrated into the MCP server, otherwise, there will be a situation where the policy is still reconciling and the routing has been published online. Providing network routing configurations in MCP server would undoubtedly be a significant workload.

If you develop your own Native Plugin or Service Registry in a third-party repository and want to use it in HTNN, you can use a patch to introduce that repository into Istio. For details, refer to this patch: https://github.com/mosn/htnn/blob/main/patch/istio/1.21/20240410-htnn-go-mod.patch.

If you want to run the HTNN control plane, you can refer to the implementation of `make e2e-prepare-controller-image` in the `e2e/` directory to see how to package the Istio embedded with HTNN into an image.

## Plugins

HTNN places data plane-running Go Plugins and their shared pkg libraries in the `plugins/` module.

If you want to write a Go Plugin in a third-party repository, you can refer to this example: https://github.com/mosn/htnn/tree/main/examples/dev_your_plugin. Go Plugins are compiled into shared libraries and deployed to the data plane. When you want to integrate a Go Plugin into the data plane, this file can serve as a template: https://github.com/mosn/htnn/blob/main/examples/dev_your_plugin/cmd/libgolang/main.go.

It is recommended to read the existing plugins' code before developing your own plugin, especially plugins that are similar to the one you're developing – at the very least, you should look at the [demo](https://github.com/mosn/htnn/tree/main/plugins/plugins/demo) plugin. Plugin code is located in the `plugins/` directory. For more information on plugin development, please refer to the [Plugin Development](./plugin_development.md) documentation.

HTNN provides a [Plugin Integration Test Framework](./plugin_integration_test_framework) that allows the testing of Go Plugin logic to run with only the data plane running. The `dev_your_plugin` example also demonstrates how to run the integration test framework in a third-party repository.

Each plugin contains two types of objects: `config` and `filter`. Where `config` is responsible for configuration management and `filter` is responsible for executing request-level logic.

### Config

In plugin development, features closely related to configuration should all be placed in `config`.

The granularity of `config` can be per-route, per-gateway, or per-consumer, depending on the scope of the plugin configuration. For example, configuring the Gateway-level `limitCountRedis`:

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

When this occurs, a `limitCountRedis.config` object is created and shared among various routes in the Gateway `default`.

The lifecycle of `config` depends on the plugin configuration itself and the object it acts upon. For example, if we configure `limitCountRedis` on route `vs`:

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

In two cases, the `limitCountRedis.config` object will be recreated:
1. Any changes to the routing under any Gateway that `vs` belongs to
2. Any changes into the specification of FilterPolicy that points to routes mentioned in the previous case

This is because new [RouteConfiguration](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route.proto#envoy-v3-api-msg-config-route-v3-routeconfiguration) in the above cases causes the `limitCountRedis.config` object to be re-created.

(TODO: Support incremental update of routes so that only changes to `vs` itself cause the object to be re-created)

As `config` is shared across multiple requests in different Envoy workers, read and write operations to `config` must be locked.

### Filter

In plugin development, features related to each request should be placed in `filter`. Each request creates a `filter` object for each plugin that is executed.

`filter` mainly defines the following methods:

1. DecodeHeaders
2. DecodeData
3. EncodeHeaders
4. EncodeData
5. OnLog

Normally, the above methods are executed from top to bottom. However, there are exceptions:

1. If there is no body, the corresponding DecodeData and EncodeData methods will not be executed.
2. Since the OnLog operation is triggered by the client interrupting the request, OnLog may execute concurrently with other methods if the client interrupts prematurely.
3. In requests like bidirectional streams, it is possible to handle the request body and upstream response at the same time, so DecodeData may execute concurrently with EncodeHeaders or EncodeData.

So, when reading and writing `filter`, there is a risk of concurrent access and lock consideration is necessary.
