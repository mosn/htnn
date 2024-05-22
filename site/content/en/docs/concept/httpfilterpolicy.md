---
title: HTTPFilterPolicy
---

Most features on gateways and service mesh revolve around the HTTP protocol, such as authentication, rate limiting, request rewriting, etc. HTNN implements these features via configuration rules provided by HTTPFilterPolicy.

Unlike some products in this area, HTNN does not use different CRDs for different HTTP relative purposes. Instead, it uses a single CRD called HTTPFilterPolicy to solve all business needs at the HTTP layer. This is because we believe the cost of multiple CRDs is too high. We even introduced a `0 CRD` [embedded mode](../embedded_mode) to reduce the cost of onboarding and maintenance.

## Structure of HTTPFilterPolicy

A typical HTTPFilterPolicy structure is as follows:

```yaml
apiVersion: htnn.mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  creationTimestamp: "2024-05-13T07:15:09Z"
  generation: 1
  name: policy
  namespace: istio-system
  resourceVersion: "158934"
  uid: 5b368582-0de3-4db0-b447-6c858b5a1305
spec:
  targetRef:
    group: networking.istio.io
    kind: VirtualService
    name: vs
    sectionName: to-httpbin
  filters:
    animal:
      config:
        pet: goldfish
    plant:
      config:
        vegetable: carrot
status:
  conditions:
  - lastTransitionTime: "2024-05-13T07:15:10Z"
    message: The policy targets non-existent resource
    observedGeneration: 1
    reason: TargetNotFound
    status: "False"
    type: Accepted
```

This HTTPFilterPolicy contains a `targetRef`, which determines the kind of resource the HTTPFilterPolicy will affect. Currently, we support the following resources:

| group                     | kind           | remarks                                                                                |
|---------------------------|----------------|----------------------------------------------------------------------------------------|
| networking.istio.io       | VirtualService |                                                                                        |
| networking.istio.io       | Gateway        | Requires control plane to enable `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS`. See details below. |
| gateway.networking.k8s.io | HTTPRoute      |                                                                                        |
| gateway.networking.k8s.io | Gateway        | Requires control plane to enable `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS`. See details below. |

`sectionName` is optional and only takes effect when `kind` is set to VirtualService. It specifies the route under VirtualService to which the policy will apply.

Currently, HTTPFilterPolicy can only affect route resources in the same namespace, and the targeted resource's Gateway must be in the same namespace as the resource.

This HTTPFilterPolicy also includes a `filters` section. Multiple plugins can be configured within `filters`, such as `animal` and `plant` in the example. The execution order of each plugin is determined by the [order specified](../../developer-guide/plugin_development#插件顺序) when the plugin is registered. Each plugin's specific configuration is located in the `config` field under the plugin name.

Like other Kubernetes resources, the HTNN control plane will modify the `status` field of the HTTPFilterPolicy to report the status of the policy. The `reason` field under `status` will be one of the following values:

| Name           | Description                                                                       |
|----------------|-----------------------------------------------------------------------------------|
| TargetNotFound | The policy's targeted resource does not exist or is invalid                       |
| Invalid        | The policy is invalid                                                             |
| Accepted       | The policy can be reconciled (but does not imply effectiveness on the data plane) |

If the policy cannot be reconciled, the specific error message will be in the `message` field.

Note: Restarting or upgrading the HTNN control plane will not actively re-validate policies that are `Invalid` (i.e., `reason` is `Invalid`). If you wish to trigger re-validation (including changing a formerly valid policy into an invalid one), you need to recreate the policy manually.

## Configuring Policies with HTTPFilterPolicy in Different Scenarios

For gateways configured by API dimension, we can define a VirtualService for each API and then define an HTTPFilterPolicy pointing to it:

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: VirtualService
  metadata:
    name: vs
    namespace: default
  spec:
    gateways:
    - default
    hosts:
    - default.local
    http:
    - match:
      - uri:
          exact: /v1/api
      route:
      - destination:
          host: httpbin
          port:
            number: 8000
- apiVersion: htnn.mosn.io/v1
  kind: HTTPFilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: VirtualService
      name: vs
    filters:
      animal:
        config:
          pet: goldfish
```

Under the same domain, different APIs will use different VirtualServices, and consequently, have different HTTPFilterPolicies.

For gateways configured by domain dimension, we can define a VirtualService for each domain, and then designate an HTTPFilterPolicy to point to it:

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: VirtualService
  metadata:
    name: vs
    namespace: default
  spec:
    gateways:
    - default
    hosts:
    - default.local
    http:
    - match:
      - uri:
          exact: /httpbin
      name: to-httpbin
      route:
      - destination:
          host: httpbin
          port:
            number: 80
    - match:
      - uri:
          prefix: /
      route:
      - destination:
          host: default
          port:
            number: 8000
- apiVersion: htnn.mosn.io/v1
  kind: HTTPFilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: VirtualService
      name: vs
    filters:
      animal:
        config:
          pet: goldfish
```

If we want to issue specific policies for specific routes under the domain, we need to specify the route's name in the HTTPFilterPolicy. For instance, in the example above, there is a route:

```yaml
  http:
    - match:
      - uri:
          exact: /httpbin
      name: to-httpbin
      route:
      - destination:
          host: httpbin
          port:
            number: 80
```

Suppose we want to issue policies to this particular `to-httpbin` route, we need to specify it in `sectionName`:

```yaml
- apiVersion: htnn.mosn.io/v1
  kind: HTTPFilterPolicy
  metadata:
    name: to-httpbin-policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: VirtualService
      name: vs
      sectionName: to-httpbin
    filters:
      animal:
        config:
          pet: cat
```

For other routes inside the VirtualService, the configuration corresponding to “pet” is goldfish. Only the “to-httpbin” route configuration is cat.

We can also set policies at the [LDS](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/lds.html) level, which you can understand as port-level configuration. To configure policies at the LDS level, you need to set the environment variable `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS` to true when starting the control plane.

Take the following configuration as an example:

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: Gateway
  metadata:
    name: httpbin-gateway
    namespace: default
  spec:
    selector:
      istio: ingressgateway
    servers:
    - hosts:
      - '*'
      port:
        name: http
        number: 80
        protocol: HTTP
    - hosts:
      - '*'
      port:
        name: https
        number: 443
        protocol: HTTPS
- apiVersion: htnn.mosn.io/v1
  kind: HTTPFilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: Gateway
      name: httpbin-gateway
    filters:
      animal:
        config:
          pet: goldfish
```

When we issue an HTTPFilterPolicy targeting a Gateway, all routes under the LDS generated by that Gateway (here referring to ports 80 and 443) will have the `animal` plugin configured.

By default, `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS` is disabled because this feature generates a corresponding ECDS for each LDS.

1. With a large number of LDS, a multitude of ECDS can cause significant overhead.
2. We cannot use ECDS to disable an HTTP filter at the LDS level. So every route under this LDS will incur the overhead of switching to the Go land.

If the following scenarios apply to you, you can enable it:

1. If you are using HTNN as a gateway.
2. The number of LDS is limited. It is better to run benchmark tests to see if the performance is expected.
3. You need plugins at the LDS level.

Although we have used Istio's CRDs in the above examples, the same configuration can be achieved using Gateway API, by just changing the contents of `targetRef`.

Plugins configured by different HTTPFilterPolicies with overlapping scopes will merge and then execute in the order specified at the time the plugins were registered. If different levels of HTTPFilterPolicy configure the same plugin, the configuration on the smaller scoped HTTPFilterPolicy will override the broader scoped configuration, namely `SectionName` > `VirtualService/HTTPRoute` > `Gateway`. If the same plugin is configured by the same level of HTTPFilterPolicy, the HTTPFilterPolicy created earliest takes precedence; if the timings are the same, they are ordered by the namespace and name of the HTTPFilterPolicy.

## Using SubPolicies to Reduce the Number of HTTPFilterPolicies

For gateways configured by domain dimension, a VirtualService could contain hundreds of routes. If each route requires its configuration, we would need to create hundreds of HTTPFilterPolicies. To reduce the load on the API server, we support targeting multiple routes with a single HTTPFilterPolicy as shown below:

```yaml
- apiVersion: networking.istio.io/v1beta1
  kind: VirtualService
  metadata:
    name: vs
    namespace: default
  spec:
    gateways:
    - default
    hosts:
    - default.local
    http:
    - match:
      - uri:
          exact: /a
      name: route
      route:
      - destination:
          host: httpbin
          port:
            number: 8000
    - match:
      - uri:
          prefix: /
      name: route2
      route:
      - destination:
          host: httpbin
          port:
            number: 8000
- apiVersion: htnn.mosn.io/v1
  kind: HTTPFilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: VirtualService
      name: vs
    subPolicies:
    - sectionName: route
      filters:
        animal:
          config:
            pet: bird
    - sectionName: route2
      filters:
        animal:
          config:
            pet: dog
```

HTTPFilterPolicy supports using the `subPolicies` field to configure policies for multiple `sectionNames` simultaneously. Both `filters` and `subPolicies` can be used together, and the merging rules for configurations are the same as when using multiple separate HTTPFilterPolicies.
