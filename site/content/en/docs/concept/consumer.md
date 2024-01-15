---
title: Consumer
---

When using an API gateway, we frequently need to apply the same authentication logic to multiple routes. Therefore, HTNN introduces the `Consumer` concept, allowing users to delegate common authentication configurations and related extension operations to a dedicated Custom Resource Definition (CRD), thereby better managing API assets.

For instance, let's assume we have a consumer named `Leo` and two routes. Due to historical reasons, the authentication parameters on these two routes are obtained from different sources — one from the url and the other from the header — as demonstrated below:

```yaml
apiVersion: mosn.io/v1
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
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
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
          source: query
---
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
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
          source: header
```

Without the Consumer layer of abstraction, each route would have to be configured with `key: Leo`. Imagine one day, for example, `Leo` is just a temporary user and we need to revoke their permissions. With the Consumer, we would only need to delete the consumer `Leo`, without any need to alter the route configurations.

Furthermore, we can configure specific plugins for the consumer. These plugins will only execute after the authentication process has passed. Take the following configuration as an example:

```yaml
apiVersion: mosn.io/v1
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
apiVersion: mosn.io/v1
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
apiVersion: mosn.io/v1
kind: HTTPFilterPolicy
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
          source: header
```

If the authentication result is for a prestigious VIP member, then the `average` configuration would be 10. If it's a regular member, then the corresponding configuration would be just 1.

Unlike consumers in Kong/APISIX, HTNN's consumers are at the `namespace` level. Consumers from different `namespaces` will only apply to the Routes within their respective `namespace` configurations (HTTPRoute, VirtualService, etc.). This design prevents consumer conflicts between different business units.