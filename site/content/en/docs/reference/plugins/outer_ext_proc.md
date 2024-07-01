---
title: Outer Ext Proc
---

## Description

The `outerExtProc` plugin allows for communication with an external service during the request processing phase by leveraging Envoy's `ext_proc` filter, using the external service's processing results to rewrite the request or response.

As Envoy uses an onion model to proxy requests, the execution order is:

1. Request starts
2. Run `outerExtProc` and other plugins in the `Outer` group
3. Run other plugins
4. Proxy to upstream
5. Handle the response with other plugins
6. Run `outerExtProc` and other plugins in the `Outer` group to handle the response
7. Request ends

## Attribute

|       |         |
| ----- | ------- |
| Type  | General |
| Order | Outer   |

## Configuration

For specific configuration fields, see the corresponding [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.29.4/api-v3/extensions/filters/http/ext_proc/v3/ext_proc.proto#envoy-v3-api-msg-extensions-filters-http-ext-proc-v3-extprocoverrides).

For the working principles of External Processing, refer to [Envoy's introduction to External Processing](https://www.envoyproxy.io/docs/envoy/v1.29.5/configuration/http/http_filters/ext_proc_filter.html).

## Usage

Suppose we have the following HTTPRoute attached to `localhost:10000`, and there is a backend server listening on port `8080`:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: default
spec:
  parentRefs:
  - name: default
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend
      port: 8080
```

We run a service in k8s with the namespace `default` and the name `processor`. It listens on port `8080` and implements the External Processing protocol of Envoy.

By applying the configuration below, we can modify the requests to `http://localhost:10000/` with the processor service's response:

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    outerExtProc:
      config:
        grpcService:
          envoyGrpc:
            clusterName: outbound|8080||processor.default.svc.cluster.local
```

Here the `clusterName` specifies the address of the target External Processing service, with the naming format `outbound|$port||$FQDN`. Before configuring the target service, ensure Istio has synchronized the service addresses to the data plane. We can use `istioctl pc cluster $data_plane_id` to query the list of accessible services.
