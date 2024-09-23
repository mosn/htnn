---
title: Service Registry
---

In k8s, istio interfaces with backend services through `Service`. However, not all services run within k8s. To support services outside of k8s, istio provides [ServiceEntry](https://istio.io/latest/docs/reference/config/networking/service-entry/) to describe the information of these services.

Manually managing `ServiceEntry` can solve some simple scenarios, but it is not suitable for direct use in production environments. HTNN offers a framework for interfacing with service discovery systems, which subscribes to services from the service discovery system and converts them into `ServiceEntry`, thereby introducing services connected to the service discovery system into HTNN's scope. This framework is called `service registry`, and users can use the existing `registry` or develop their own `registry` just like developing a plugin.

Currently, HTNN has provided an implementation for Nacos V1, and we welcome contributors to submit code to support more service discovery systems.

Relevant links:

* [How to develop a registry](../developer-guide/registry_development.md)
* [Existing registry documentation](../reference/registries)
