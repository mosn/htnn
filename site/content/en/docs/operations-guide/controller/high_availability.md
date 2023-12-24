---
title: High Availability
---

## Readness & Liveness Probe

The controller provides `0.0.0.0:10081/readyz` as a readiness probe and `0.0.0.0:10081/healthz` as a liveness probe. The address of the health check can be modified by specifying the `--health-probe-bind-address $another_addr` option at startup, e.g. `--health-probe-bind-address :11081`.

```
$ curl 'http://0.0.0.0:10081/readyz' -i
HTTP/1.1 200 OK
...
Content-Length: 2

ok
$ curl 'http://0.0.0.0:10081/healthz' -i
HTTP/1.1 200 OK
...
Content-Length: 2

ok
```

## Leader Election

If `--leader-elect` option is provided at startup, the controller will do leader election. If multiple replicas are deployed in the same namespace, only the elected leader is able to reconcile. Therefore, we can have multiple replicas to improve the high availability. A new election will be started 15 seconds after the leader is gone.