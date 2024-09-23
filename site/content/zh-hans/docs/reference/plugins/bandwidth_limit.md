---
title: Bandwidth Limit
---

## 说明

`bandwidthLimit` 插件通过利用 Envoy 的 `bandwidth_limit` 过滤器限制数据流的最大带宽。注意该限制只涉及请求体或响应体。

## 属性

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Outer   |

## 配置

请参阅相应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/http/http_filters/bandwidth_limit_filter)。

## 用法

假设我们有下面附加到 `localhost:10000` 的 HTTPRoute，并且有一个后端服务器监听端口 `8080`：

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
    - name: backend
      port: 8080
```

通过应用下面的配置，发送到 `http://localhost:10000/` 的请求的上传速度将被限制在大约 1kb/s：

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    bandwidthLimit:
      config:
        statPrefix: policy_bandwidth_limit
        enableMode: REQUEST
        limitKbps: 10
        fillInterval: 0.02s
```

带宽限制并不是很精准。`fillInterval` 越小，精确度越高。

为了测试插件效果，我们让监听端口 `8080` 的后端服务器返回所有收到的请求。让我们用 [bombardier](https://pkg.go.dev/github.com/codesenberg/bombardier) 试一下：

```shell
$ bombardier -m POST -f go.sum -c 10 -t 180s -d 60s -l http://localhost:10000/
Bombarding http://localhost:10000/ for 1m0s using 10 connection(s)
[======================================================================================================================================================] 1m0s
Done!
Statistics        Avg      Stdev        Max
  Reqs/sec         0.15       2.77      52.57
  Latency        40.35s     15.50s      1.29m
  Latency Distribution
     50%     38.35s
     75%     45.28s
     90%      0.99m
     95%      1.15m
     99%      1.29m
  HTTP codes:
    1xx - 0, 2xx - 19, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:    20.11KB/s
```

由于上行带宽被限制在 10KB/s，而下行带宽没有限制，bombardier 报告了整体带宽是 20.11KB/s。
