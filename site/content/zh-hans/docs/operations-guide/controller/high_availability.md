---
title: 高可用
---

## Readness & Liveness 探针

控制器提供 `0.0.0.0:10081/readyz` 作为 readiness 探针，以及 `0.0.0.0:10081/healthz` 作为 liveness 探针。可以通过在启动时指定 `--health-probe-bind-address $another_addr` 选项来修改探针的地址，例如 `--health-probe-bind-address :11081`。

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

## 领导者选举

如果在启动时提供了 `--leader-elect` 选项，控制器将进行领导者选举。如果在同一命名空间中部署了多个副本，只有被选举出的领导者能够调和 CRD。因此，我们可以部署多个副本以增强高可用性。当被选为领导者的实例消失了，15 秒后将开始新的选举。
