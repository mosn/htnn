---
title: FilterPolicy
---

Most businesses on gateways and service mesh revolve around network protocols, such as authentication, rate limiting, request rewriting, etc. HTNN abstracts these needs and uses FilterPolicy to express specific configuration rules.

Unlike some similar products, HTNN does not use different CRDs for different business categories, but unifies all policy-level business needs using a single CRD, FilterPolicy. This is because we feel the cost of multiple CRDs is too high. We even introduced the `0 CRD` [embedded mode](./embedded_mode.md) to reduce the cost of integration and maintenance.

## Structure of FilterPolicy

A typical FilterPolicy structure is as follows:

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
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

This FilterPolicy contains a `targetRef`, which determines the kind of resource the FilterPolicy will affect. Currently, we support the following resources:

| group                     | kind           | remarks                                                                                |
|---------------------------|----------------|----------------------------------------------------------------------------------------|
| networking.istio.io       | VirtualService |                                                                                        |
| networking.istio.io       | Gateway        | Requires control plane to enable `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS`. See details below. |
| gateway.networking.k8s.io | HTTPRoute      |                                                                                        |
| gateway.networking.k8s.io | Gateway        | Requires control plane to enable `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS`. See details below. |

The `sectionName` field is optional and is only effective when the `kind` is set to either VirtualService or Gateway.

* When it applies to VirtualService, it can be used to specify which route under the VirtualService it takes effect on. At this time, the sectionName needs to match the name field of a route under the VirtualService. Note that if multiple VirtualServices with the same domain name set routes with the same name, Istio will eventually generate multiple routes with the same name for that domain, leading to the FilterPolicy actually hitting another route with the same name on other VirtualServices. Therefore, for different VirtualServices under the same domain name, routes with the same name should be avoided.
* When it applies to Gateway, it can be used to specify which particular Server or Listener under Gateway it will be effective for. In this case, `sectionName` must match the `name` field of a Server under the istio Gateway or a Listener under the k8s Gateway. Note that since the policy at the Gateway level currently only applies at the port level, it is, in effect, applicable to the port where the matched Server or Listener is located.

For specific examples of using `sectionName`, see the following.

Currently, FilterPolicy can only affect route resources in the same namespace, and the targeted resource's Gateway must be in the same namespace as the resource.

This FilterPolicy also includes a `filters` section. Multiple plugins can be configured within `filters`, such as `animal` and `plant` in the example. The execution order of each plugin is determined by the [order specified](../developer-guide/plugin_development.md#plugin-order) when the plugin is registered. Each plugin's specific configuration is located in the `config` field under the plugin name.

Like other Kubernetes resources, the HTNN control plane will modify the `status` field of the FilterPolicy to report the status of the policy. The `reason` field under `status` will be one of the following values:

| Name           | Description                                                                       |
|----------------|-----------------------------------------------------------------------------------|
| TargetNotFound | The policy's targeted resource does not exist or is invalid                       |
| Invalid        | The policy is invalid                                                             |
| Accepted       | The policy can be reconciled (but does not imply effectiveness on the data plane) |

If the policy cannot be reconciled, the specific error message will be in the `message` field.

Note: Restarting or upgrading the HTNN control plane will not actively re-validate policies that are `Invalid` (i.e., `reason` is `Invalid`). If you wish to trigger re-validation (including changing a formerly valid policy into an invalid one), you need to recreate the policy manually.

## Configuring Policies with FilterPolicy in Different Scenarios

For gateways configured by API dimension, we can define a VirtualService for each API and then define a FilterPolicy pointing to it:

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
  kind: FilterPolicy
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

Under the same domain, different APIs will use different VirtualServices, and consequently, have different FilterPolicies.

For gateways configured by domain dimension, we can define a VirtualService for each domain, and then designate a FilterPolicy to point to it:

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
  kind: FilterPolicy
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

If we want to issue specific policies for specific routes under the domain, we need to specify the route's name in the FilterPolicy. For instance, in the example above, there is a route:

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
  kind: FilterPolicy
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
        name: port-http
        number: 80
        protocol: HTTP
    - hosts:
      - '*'
      name: https
      port:
        name: port-https
        number: 443
        protocol: HTTPS
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
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

When we issue a FilterPolicy targeting a Gateway, all routes under the LDS generated by that Gateway (here referring to ports 80 and 443) will have the `animal` plugin configured.

We can also use `sectionName` to specify the configuration to be issued to port 443:

```yaml
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: Gateway
      name: httpbin-gateway
      sectionName: https
    filters:
      animal:
        config:
          pet: cat
```

Note that as we currently only support port-level Gateway configurations, the behavior for policies configured for multiple sections of the same port is undefined. In the future, we may support more granular configurations.

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
      - '*.test.com'
      name: test
      port:
        name: port-https
        number: 443
        protocol: HTTPS
    - hosts:
      - '*.example.com'
      name: example
      port:
        name: port-https
        number: 443
        protocol: HTTPS
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: Gateway
      name: httpbin-gateway
      sectionName: test
    filters:
      animal:
        config:
          pet: goldfish
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: networking.istio.io
      kind: Gateway
      name: httpbin-gateway
      sectionName: example
    filters:
      animal:
        config:
          pet: cat
```

As a result, the `pet` on port 443 could potentially be either goldfish or cat. We do not guarantee behavior in such cases at the moment.

By default, `HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS` is disabled because this feature generates a corresponding ECDS for each LDS.

1. With a large number of LDS, a multitude of ECDS can cause significant overhead.
2. We cannot use ECDS to disable an HTTP filter at the LDS level. So every route under this LDS will incur the overhead of switching to the Go land.

If the following scenarios apply to you, you can enable it:

1. If you are using HTNN as a gateway.
2. The number of LDS is limited. It is better to run benchmark tests to see if the performance is expected.
3. You need plugins at the LDS level.

Although we have used Istio's CRDs in the above examples, the same configuration can be achieved using Gateway API, by just changing the contents of `targetRef`.

For example, to issue a policy for a particular Listener of a Gateway API's Gateway resource, you could configure as follows:

```yaml
- apiVersion: gateway.networking.k8s.io/v1
  kind: Gateway
  metadata:
    name: gw
    namespace: default
  spec
    gatewayClassName: istio
    listeners:
    - name: http
      port: 80
      protocol: HTTP
    - name: http2
      port: 8080
      protocol: HTTP
- apiVersion: htnn.mosn.io/v1
  kind: FilterPolicy
  metadata:
    name: policy
    namespace: default
  spec:
    targetRef:
      group: gateway.networking.k8s.io
      kind: Gateway
      name: gw
      sectionName: http
    filters:
      limitReq:
        config:
          average: 1
```

Plugins configured by different FilterPolicies with overlapping scopes will merge and then execute in the order specified at the time the plugins were registered. If different levels of FilterPolicy configure the same plugin, the configuration on the smaller scoped FilterPolicy will override the broader scoped configuration, namely `SectionName` > `VirtualService/HTTPRoute` > `Gateway`.

If the same plugin is configured by the same level of FilterPolicy, then the FilterPolicy with the earlier creation time takes precedence (the creation time depends on the k8s auto-popopulated creationTimestamp field); if the times are the same, then the FilterPolicy is sorted by its namespace and name. Since FilterPolicy in embedded mode doesn't have auto-populated creationTimestamp field, FilterPolicy in embedded mode will always have the highest priority.

## The Relationship between FilterPolicy and Plugins

FilterPolicy is simply the carrier for plugins. HTNN's plugins can be divided into two categories:

* Go plugins that run on the data plane
* Plugins that run on the control plane to generate Envoy configurations, which we call Native plugins

Depending on their location, Native plugins can be further divided into the following categories:

* HTTP Native plugins, which affect HTTP filters
* Network Native plugins, which affect Network filters
* Listener Native plugins, which affect Listeners

In the documentation for each plugin, we have indicated its category. In the "Attribute" section, if `Order` is:

* `Listener`, it is a Listener Native plugin
* `Network`, it is a Network Native plugin
* `Outer` or `Inner`, it is an HTTP Native plugin
* The rest are Go plugins

Which plugins can be configured on a FilterPolicy depends on the target resource type in `TargetRef`, as shown in the table below:

| Plugin Type             | Configured on Gateway | Configured on Route |
|-------------------------|-----------------------|---------------------|
| Go plugins              | Supported             | Supported           |
| HTTP Native plugins     | Pending support       | Supported           |
| Network Native plugins  | Supported             | Not supported       |
| Listener Native plugins | Supported             | Not supported       |

## Using SubPolicies to Reduce the Number of FilterPolicies

For gateways configured by domain dimension, a VirtualService could contain hundreds of routes. If each route requires its configuration, we would need to create hundreds of FilterPolicies. To reduce the load on the API server, we support targeting multiple routes with a single FilterPolicy as shown below:

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
  kind: FilterPolicy
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

FilterPolicy supports using the `subPolicies` field to configure policies for multiple `sectionNames` simultaneously. Both `filters` and `subPolicies` can be used together, and the merging rules for configurations are the same as when using multiple separate FilterPolicies.

Note that `subPolicies` currently only supports VirtualService.
