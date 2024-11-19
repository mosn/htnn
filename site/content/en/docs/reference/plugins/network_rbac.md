---
title: Network RBAC
---

## Description

The `networkRBAC` plugin supports configuring rules to determine whether to deny requests based on layer 4 network attributes (IP, SNI, etc).

## Attribute

|        |              |
|--------|--------------|
| Type   | Authz        |
| Order  | Network      |
| Status | Experimental |

## Configuration

Please refer to the corresponding [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/listeners/network_filters/rbac_filter).

## Usage

Assume we have the following Gateway listening on `localhost:10000`:

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

The configuration below will deny all requests from `127.0.0.1` and allow all other requests:

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

Let's try it out:

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

If we modify the policy to no longer ban requests from `127.0.0.1`:

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

Now, the connection will be successful:

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
