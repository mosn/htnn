---
title: Istio
---

## Introduction

Istio is the main component of the HTNN control plane. In addition, HTNN also implements its own controller to reconcile CRDs such as HTTPFilterPolicy. HTNN combines its controller with Istio through some patches. These patches include the following features:

* Watching HTNN's own CRDs.
* In the `PushContext`, detect if there is a need to invoke the HTNN controller to reconcile the current state.
* The HTNN controller generates Istio resources such as EnvoyFilter and writes them into Istio's `ConfigStore`. The configuration pushed to Envoy by Istio will include these generated resources.

Additionally, HTNN has also backported some of Istio's latest bugfixes to the Istio version currently used by HTNN.

HTNN's distribution of Istio is 100% compatible with the official Istio release. All features of Istio are available on HTNN's Istio distribution. Compared to the original Istio, HTNN has the following changes:

* Using its own control plane and data plane images.
* Added some environment variables, see the table below.
* Added its own CRDs, as well as RBAC and webhook configurations around these CRDs.

## HTNN-Related Environment Variables

| Name                               | Type    | Default Value     | Description                                                                                                                                                                                |
|------------------------------------|---------|-------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| PILOT_ENABLE_HTNN                  | Boolean | false             | If enabled, Pilot will listen for HTNN resources.                                                                                                                                          |
| PILOT_ENABLE_HTNN_STATUS           | Boolean | false             | If set to true, we will report status information to HTNN resources.                                                                                                                       |
| PILOT_SCOPE_GATEWAY_TO_NAMESPACE   | Boolean | false             | This environment variable is set to true in HTNN. We assume the workload's namespace is the same as the gateway's namespace to reduce the complexity of managing namespaces for workloads. |
| HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS    | Boolean | false             | Enables the capability to deploy LDS plugins via ECDS.                                                                                                                                     |
| HTNN_ENVOY_GO_SO_PATH              | String  | /etc/libgolang.so | The path to the Go shared library in the data plane image.                                                                                                                                 |
| HTNN_ENABLE_NATIVE_PLUGIN          | Boolean | true              | Allows configuring Native plugins via the HTNN controller.                                                                                                                                 |
| HTNN_ENABLE_EMBEDDED_MODE          | Boolean | true              | Enables [embedded mode](../../concept/embedded_mode).                                                                                                                                      |
| HTNN_USE_WILDCARD_IPV6_IN_LDS_NAME | Boolean | false             | Use a wildcard IPv6 address as the default prefix in the LDS name. Turn this on if your gateway is listening to an IPv6 address by default.                                                |
