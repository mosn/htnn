---
title: Embedded Mode
---

In some scenarios, issuing route configuration and policy configuration separately can present challenges:

* There can be inconsistencies between route configurations and policy configurations. For example, Istio may have watched the route configuration, but the policy configuration could have been delayed due to API server throttling. Enforcing a specific order of operations for routes and policies can be complex.
* For scenarios where policy entirely depends on route configuration, users prefer to express both routes and plugin configurations using a single CRD, similar to traditional gateways, reducing understanding and maintenance costs.

To solve these problems, HTNN offers the Embedded Mode.

Suppose we have an HTTPFilterPolicy like this:

```yaml
apiVersion: htnn.mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: policy
  namespace: default
spec:
  filters:
    animal:
      config:
        kind: cat
  subPolicies:
  - sectionName: route1
    filters:
      animal:
        config:
          kind: goldfish
  - sectionName: route2
    filters:
      animal:
        config:
          kind: catfish
```

Since this resource is only configured via annotation and the target is fixed, there's no need to additionally specify a `targetRef`.

When using it, users serialize the above resource into JSON and place it in the `htnn.mosn.io/httpfilterpolicy` annotation as shown below:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: vs
  namespace: default
  annotations:
    htnn.mosn.io/httpfilterpolicy: |
      {"apiVersion":"htnn.mosn.io/v1","kind":"HTTPFilterPolicy","metadata":{"name":"policy","namespace":"default"},"spec":{"filters":{"animal":{"config":{"kind":"cat"}}},"subPolicies":[{"sectionName":"route1","filters":{"animal":{"config":{"kind":"goldfish"}}}},{"sectionName":"route2","filters":{"animal":{"config":{"kind":"catfish"}}}]}}
spec:
  gateways:
  - default
  hosts:
  - default.local
  http:
  - match:
    - uri:
        prefix: /
    name: route
    route:
    - destination:
        host: httpbin
        port:
          number: 8000
```

When the control plane receives the VirtualService, it will check if there's an HTTPFilterPolicy in its annotation `htnn.mosn.io/httpfilterpolicy`. If present, it's as if both HTTPFilterPolicy and the corresponding VirtualService were received. Therefore, when issuing routes and policies, only the VirtualService needs to be issued. Unlike the Ingress annotations, the HTTPFilterPolicy here will still participate in policy merging, so users can still specify a higher-level HTTPFilterPolicy (e.g., affecting the entire `Gateway`) to add additional plugins.

Note that Embedded Mode currently only supports embedding the HTTPFilterPolicy into VirtualService. Embedding HTTPFilterPolicy in Istio Gateway is also supports, but it requires the control plane to enable `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS`.
