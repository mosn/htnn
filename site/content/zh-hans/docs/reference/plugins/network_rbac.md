---
title: Network RBAC
---

## 说明

`networkRBAC` 插件支持配置规则，根据四层网络特征（IP、SNI 等）决定是否拒绝请求。

## 属性

|        |              |
|--------|--------------|
| Type   | Authz        |
| Order  | Network      |
| Status | Experimental |

## 配置

请查阅对应的 [Envoy 文档](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/listeners/network_filters/rbac_filter)。

## 用法

假设我们有以下的 Gateway 在 `localhost:10000` 上监听：

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: default
spec:
  gatewayClassName: istio
  listeners:
  - name: default
    hostname: "*"
    port: 10000
    protocol: HTTP
```

下面的配置将拒绝所有来自 `127.0.0.1` 的请求，并放行除此之外的所有请求：

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: default
  filters:
    networkRBAC:
      config:
        statPrefix: network_rbac
        matcher:
          matcher_tree:
            input:
              name: envoy.matching.inputs.source_ip
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceIPInput
            custom_match:
              name: ip-matcher
              typed_config:
                "@type": type.googleapis.com/xds.type.matcher.v3.IPMatcher
                range_matchers:
                - ranges:
                  - address_prefix: 127.0.0.1
                    prefix_len: 32
                  on_match:
                    action:
                      name: envoy.filters.rbac.action
                      typed_config:
                        "@type": type.googleapis.com/envoy.config.rbac.v3.Action
                        name: localhost
                        action: DENY
                # match-all action
                - ranges:
                  - address_prefix: 0.0.0.0
                    prefix_len: 0
                  on_match:
                    action:
                      name: envoy.filters.rbac.action
                      typed_config:
                        "@type": type.googleapis.com/envoy.config.rbac.v3.Action
                        name: match-all
                        action: ALLOW
```

让我们试一下：

```shell
$ curl -I http://localhost:10000/ -v
*   Trying 127.0.0.1:10000...
* Connected to localhost (127.0.0.1) port 10000 (#0)
> HEAD / HTTP/1.1
> Host: localhost:10000
> User-Agent: curl/7.87.0
> Accept: */*
>
* Empty reply from server
* Closing connection 0
curl: (52) Empty reply from server
```

如果我们修改策略，比如不再禁止来自 `127.0.0.1` 的请求：

```yaml
            ...
            - ranges:
                  - address_prefix: 127.0.0.1
                    prefix_len: 32
                  on_match:
                    action:
                      name: envoy.filters.rbac.action
                      typed_config:
                        "@type": type.googleapis.com/envoy.config.rbac.v3.Action
                        name: localhost
                        action: ALLOW
            ...
```

这时候就能通了：

```shell
$ curl -I http://localhost:10000/ -v
*   Trying 127.0.0.1:10000...
* Connected to localhost (127.0.0.1) port 10000 (#0)
> HEAD / HTTP/1.1
> Host: localhost:10000
> User-Agent: curl/7.87.0
> Accept: */*
>
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
HTTP/1.1 200 OK
...
```
