# Observability

To operate the controller properly, it provides the observability below.

## Log & Trace

By default, the controller writes its log to stderr, in the format below:

```
2023-12-09T11:47:05.134+0800    info    reconcile       {"controller": "httpfilterpolicy", "controllerGroup": "mosn.io", "controllerKind": "HTTPFilterPolicy", "HTTPFilterPolicy": {"name":"policy","namespace":"default"}, "namespace": "default", "name": "policy", "reconcileID": "fd710b63-4416-4471-bb3d-5e081f03aa86"}
```

Most of the logs in the reconciliation will contain the `reconcileID` field, so we can use it as the trace ID for each reconciliation.

If you prefer a JSON format log, you can pass `--log-encoder json` argument when starting the controller. Now the log will look like this:

```
{"level":"info","ts":"2023-12-09T11:47:26.863+0800","msg":"reconcile","controller":"httpfilterpolicy","controllerGroup":"mosn.io","controllerKind":"HTTPFilterPolicy","HTTPFilterPolicy":{"name":"policy","namespace":"default"},"namespace":"default","name":"policy","reconcileID":"3120c72c-68ba-4e8b-b661-f68ac1fda49b"}
```

## Metrics

By default, the controller exposes metrics via `http://127.0.0.1:10080/metrics`, in the Prometheus format:

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

The provided metrics can be divided into the following categories according to their source:

* controller-runtime: see https://book.kubebuilder.io/reference/metrics-reference.
* prometheus client-go: see https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/collectors
* htnn: see below

It's recommended to watch the metrics below:

* `controller_runtime_reconcile_errors_total` is a counter that tracks how many reconciliation errors happened.
* `controller_runtime_reconcile_time_seconds_bucket` can be used to detect slow reconciles.
* `workqueue_queue_duration_seconds` can be used to detect if the controller is overloaded, or frequently requeuing due to error.
* `process_cpu_seconds_total` and `process_resident_memory_bytes` can be used to build resource monitor.

, and the metrics provided by htnn:

* `htnn_translate_duration_seconds_bucket` records the translation part of the reconciliation. A reconciliation can be divided into three parts: building resources from the local cache, translation, and writing to k8s API server. This histogram tracks the time spent in translation.
