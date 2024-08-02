---
title: Consumer
---

When using an API gateway, we frequently need to apply the same authentication logic to multiple routes. Therefore, HTNN introduces the `Consumer` concept, allowing users to delegate common authentication configurations and related extension operations to a dedicated Custom Resource Definition (CRD), thereby better managing API assets.

For instance, let's assume we have a consumer named `Leo` and two routes. Due to historical reasons, the authentication parameters on these two routes are obtained from different sources — one from the url and the other from the header — as demonstrated below:

```yaml
apiVersion: htnn.mosn.io/v1
kind: Consumer
metadata:
  name: leo
spec:
  auth:
    keyAuth:
      config:
        key: Leo
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: alpha
spec:
  parentRefs:
  - name: default
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend1
      port: 8080
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: beta
spec:
  parentRefs:
  - name: default
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend2
      port: 8081
---
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: alpha
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: alpha
  filters:
    keyAuth:
      keys:
        - name: ak
          source: QUERY
---
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: beta
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: beta
  filters:
    keyAuth:
      keys:
        - name: Authorization
          source: HEADER
```

Without the Consumer layer of abstraction, each route would have to be configured with `key: Leo`. Imagine one day, for example, `Leo` is just a temporary user and we need to revoke their permissions. With the Consumer, we would only need to delete the consumer `Leo`, without any need to alter the route configurations.

[Consumer plugins](../developer-guide/plugin_development.md) can be configured within the `auth` field of a Consumer. There are two types of configurations for each Consumer plugin: one is set on the Route, which specifies the source of authentication parameters, such as the `keys` in the previously mentioned FilterPolicy. The second type is configured on the Consumer, which determines the matching authentication parameters for that specific Consumer, as exemplified by the `key` in the above Consumer.

Each Consumer plugin configured on the Route will proceed through the following steps:

1. Retrieve the authentication parameters from the specified source.
2. If not found, continue to the next plugin.
3. If found, match it against a Consumer.
   1. If the match is unsuccessful, return a 401 HTTP status code.
   2. If the match is successful, move on to the next plugin.

If no Consumer is matched after all the Consumer plugins have been executed, a 401 HTTP status code will be returned.

In addition to that, we can configure additional plugins for consumers under the `filters` field. These plugins are only executed after the consumer has been authenticated. Take the following configuration as an example:

```yaml
apiVersion: htnn.mosn.io/v1
kind: Consumer
metadata:
  name: vip
spec:
  auth:
    keyAuth:
      config:
        key: vip
  filters:
    limitReq:
      config:
        average: 10
---
apiVersion: htnn.mosn.io/v1
kind: Consumer
metadata:
  name: member
spec:
  auth:
    keyAuth:
      config:
        key: member
  filters:
    limitReq:
      config:
        average: 1
---
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: beta
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: beta
  filters:
    keyAuth:
      keys:
        - name: Authorization
          source: HEADER
```

If the authentication result is for a prestigious VIP member, then the `average` configuration would be 10. If it's a regular member, then the corresponding configuration would be just 1.

All plugins implemented in Go and set to execute after the authentication order can be configured as additional plugins for consumers.

Unlike consumers in some gateways, HTNN's consumers are at the `namespace` level. Consumers from different `namespaces` will only apply to the Routes within their respective `namespace` configurations (HTTPRoute, VirtualService, etc.). This design prevents consumer conflicts between different business units.
