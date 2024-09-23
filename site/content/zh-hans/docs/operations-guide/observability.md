---
title: 可观测性
---

HTNN 100% 兼容 istio 和 Envoy，所以我们可以使用 istio 的[可观测性功能](https://istio.io/latest/docs/concepts/observability/)。

除此之外，HTNN 还提供了一些额外的可观测性功能。

注意：以下功能依赖 istio 开启 debug 接口和 prometheus 指标。默认情况下它们都是启用的。

## Log

HTNN 控制面额外添加的功能都会使用 `htnn` 这一个 logger。你可以通过 [ControlZ](https://istio.io/latest/docs/ops/diagnostic-tools/controlz/) 动态调整日志级别。通过将 `htnn` 的日志级别设置为 `debug`，您可以查看到整个 reconciliation 的过程。

HTNN 数据面基于 Go 开发的功能的日志都会使用 `golang` 这一个 logger。你可以通过 [Envoy Admin API](https://www.envoyproxy.io/docs/envoy/latest/operations/admin#post--logging) 或者 `istioctl pc log $pod_name --level golang:debug` 来动态调整日志级别。

## Metrics

HTNN 控制面额外增加了下面的指标：

| 名称                                            | 类型      | 说明                                                        |
|-------------------------------------------------|-----------|-------------------------------------------------------------|
| htnn_filterpolicy_reconcile_duration_seconds    | histogram | HTNN 调和 FilterPolicy 的耗时，单位为秒。                   |
| htnn_filterpolicy_translate_duration_seconds    | histogram | HTNN 调和 FilterPolicy 过程中花在翻译 FilterPolicy 的时间。 |
| htnn_consumer_reconcile_duration_seconds        | histogram | HTNN 调和 Consumer 的耗时，单位为秒。                       |
| htnn_serviceregistry_reconcile_duration_seconds | histogram | HTNN 调和 ServiceRegistry 的耗时，单位为秒。                |

默认访问 istio 的 prometheus 端口 `127.0.0.1:15014/metrics` 即可获取这些指标。注意如果某项指标没有数据，则不会出现。

## Debug

HTNN 控制面调和时生成的 EnvoyFilter 和 ServiceEntry 都可以通过 istio 自己的 configz 接口获取。例如执行 `kubectl exec -it istiod-xxx -- curl 127.0.0.1:8080/debug/configz | jq` 可以看到：

```json
{
  "kind": "EnvoyFilter",
  "apiVersion": "networking.istio.io/v1alpha3",
  "metadata": {
    "name": "htnn-http-filter",
    "namespace": "istio-system",
    "resourceVersion": "52795",
    "creationTimestamp": "2024-05-10T10:38:02Z",
    "labels": {
      "htnn.mosn.io/created-by": "FilterPolicy"
    },
    "annotations": {
      "htnn.mosn.io/info": "{\"filterpolicies\":[\"nodesentry/policy\"]}"
    }
  },
  ...
},
...
```

生成的 EnvoyFilter 会打上 "htnn.mosn.io/created-by" 的 label，标记它是由哪种资源生成的。另外还有一个 annotation "htnn.mosn.io/info"，其中包含下面的字段：

* `filterpolicies`: 生成该 EnvoyFilter 的策略，命名方式为 `$namespace/$name`。
