---
title: Bandwidth Limit
---

## Description

The `bandwidthLimit` plugin restricts the maximum bandwidth of the data stream by leveraging Envoy's `bandwidth_limit` filter. Note that this limitation only applies to the request or response body.

## Attribute

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Outer   |

## Configuration

See the corresponding [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.29.2/configuration/http/http_filters/bandwidth_limit_filter).

## Usage

Assumed we have the HTTPRoute below attached to `localhost:10000`, and a backend server listening to port `8080`:

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
    - name: backend
      port: 8080
```

By applying the configuration below, the upload speed of requests sent to `http://localhost:10000/` will be limited to approximately 10kb/s:

```yaml
apiVersion: htnn.mosn.io/v1
kind: HTTPFilterPolicy
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

The bandwidth limit is not very precise. The smaller the `fillInterval`, the higher the accuracy.

To test the effect of the plugin, we have the backend server listening on port `8080` which echoes all received requests. Let's try it with [bombardier](https://pkg.go.dev/github.com/codesenberg/bombardier):

```
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

Since the upstream bandwidth is limited to 10KB/s and the downstream bandwidth is not restricted, bombardier reported an overall bandwidth of 20.11KB/s.
