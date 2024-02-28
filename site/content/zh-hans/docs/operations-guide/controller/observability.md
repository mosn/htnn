---
title: 可观测性
---
为了确保正确运行，控制器提供了以下可观测性功能。

## Log & Trace

默认情况下，控制器将其日志写入 stderr，格式如下：

```
2023-12-18T16:47:06.628+0800	info	reconcile	{"controller": "httpfilterpolicy", "namespace": "", "name": "httpfilterpolicies", "reconcileID": "8e74b3ab-e223-4e40-a568-81c8179f9c5a"}
```

大多数调和过程中的日志都会包含 `reconcileID` 字段，因此我们可以使用它作为每次调和的追踪 ID。

如果您更喜欢 JSON 格式的日志，可以在启动控制器时传递 `--log-encoder json` 选项。现在日志将显示如下：

```
{"level":"info","ts":"2023-12-18T16:49:16.634+0800","msg":"reconcile","controller":"httpfilterpolicy","namespace":"","name":"httpfilterpolicies","reconcileID":"55230dc9-b035-44e9-a2d7-3a84ecc4da50"}
```

## Metrics

默认情况下，控制器通过 `http://0.0.0.0:10080/metrics` 暴露指标，可以通过在启动控制器时传递 `--metrics-bind-address $another_addr` 选项（例如，`--metrics-bind-address :11080`）来更改。

指标采用 Prometheus 格式：

```
...
# HELP controller_runtime_reconcile_errors_total Total number of reconciliation errors per controller
# TYPE controller_runtime_reconcile_errors_total counter
controller_runtime_reconcile_errors_total{controller="httpfilterpolicy"} 0
# HELP controller_runtime_reconcile_time_seconds Length of time per reconciliation per controller
# TYPE controller_runtime_reconcile_time_seconds histogram
controller_runtime_reconcile_time_seconds_bucket{controller="httpfilterpolicy",le="0.005"} 1
controller_runtime_reconcile_time_seconds_bucket{controller="httpfilterpolicy",le="0.01"} 1
...
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 58
...
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 1.604542464e+09
...
# HELP htnn_translate_duration_seconds How long in seconds translate in a batch.
# TYPE htnn_translate_duration_seconds histogram
htnn_translate_duration_seconds_bucket{controller="httpfilterpolicy",le="0.0001"} 2
...
```

根据它们的来源，所提供的指标可以分为以下几类：

* controller-runtime: 参见 https://book.kubebuilder.io/reference/metrics-reference。
* prometheus client-go: 参见 https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/collectors
* HTNN: 见下文

建议关注以下指标：

* `controller_runtime_reconcile_errors_total` 是一个计数器，用于跟踪发生的调和错误数。
* `controller_runtime_reconcile_time_seconds_bucket` 可用于发现调和过程是否变慢。
* `workqueue_queue_duration_seconds` 可用于检测控制器是否过载，或是否因错误而频繁重新排队。
* `process_cpu_seconds_total` 和 `process_resident_memory_bytes` 可用于构建资源监控。

以及 HTNN 提供的指标：

* `htnn_translate_duration_seconds_bucket` 记录了调和过程中的翻译部分。调和可以分为三部分：从本地缓存构建翻译所需的状态、翻译和写入 K8S API 服务器。这个直方图跟踪了翻译所花费的时间。

## 性能分析

默认情况下，控制器通过 `http://127.0.0.1:10082/debug/pprof` 启用 [pprof](http://golang.org/pkg/net/http/pprof/)，可以通过在启动控制器时传递 `--pprof-bind-address $another_addr` 选项（例如，`--pprof-bind-address 127.0.0.1:11082`）来更改。传递 `--pprof-bind-address 0` 可以禁用 pprof。
