---
title: Overview
---

## What is HTNN

HTNN (Hyper Trust-Native Network) is a cloud-native L3&L4&L7 Cross-Layer global network solution product.

![HTNN](/images/HTNN.png)

Currently, only a part of it, the L7 north-south ingress gateway, is open-sourced. For simplicity, this document uses HTNN to refer to the HTNN L7 north-south ingress gateway.

## Why Open Source HTNN

HTNN originated from MOSN and has developed and integrated the MoE (MOSN on Envoy) architecture over the years, combining high performance and high research and development efficiency. It can enjoy the high-performance foundations of Envoy and the cloud-native ecosystem, and also inherit the research and development efficiency and developer ecosystem of Golang from MOSN.

HTNN is a gateway product that has been promoted and implemented within Ant Group, relying on the open-source Istio/Envoy and the MoE architecture. It has accumulated many enterprise-level capabilities, plugin platforms, productization, multi-cluster, change control/auditing, and improvements in stability and observability.

By leveraging open source and contributing back to open source, open-sourcing HTNN aims to enable the community to enjoy the product capabilities we have built and hope that the community and us can work together to build an advanced gateway product.

## Golang Plugin Extension

As a language in new generation, Golang has advantages such as high performance, simplicity, strong concurrency support, and a rich development ecosystem. It has been widely used in areas such as networking and cloud-native.

Since developing MOSN, we have enjoyed the tremendous advantages of Golang. Therefore, we have also provided Golang extension capabilities for Envoy (which has been merged into the Envoy mainline), and have established the foundation of the MoE architecture.

Among the various extension mechanisms of Envoy, including C++, Lua, Ext-proc, Wasm, and Golang, the Golang development ecosystem is more attractive. In the open-source world, there are abundant Golang libraries that we can directly use for developing HTNN plugins, enabling efficient customization and development of various plugins.

Although Wasm can also be generated from Golang, it has many limitations due to the maturity of Wasm, and the performance is not ideal. Ext-proc can also be implemented in Golang for external processing services, but the inter-process communication introduced will result in relatively poor processing performance and additional service management overhead.

In the gateway scenario, there are numerous long-tail customization requirements. Having a user-friendly and reliable extension mechanism is crucial. Therefore, HTNN provides a Golang plugin extension platform, allowing developers to use the full-featured Golang to develop gateway plugins, including reusing various mature libraries from the existing Golang ecosystem, making it easy to address various traffic control requirements.

## HTNN Architecture

HTNN is a fully open-source product, from the data plane to the control plane, to the console and Dashboard (Coming soon), we will open-source everything.

![HTNN-architecture](/images/HTNN-architecture.png)

The entire architecture is a bottom-up abstraction, with stronger generality at lower levels and more specific scenarios as it goes up.

Therefore, our full-stack open-source approach ensures that each layer is standard and not forced to bind, and components can be used as needed.

## Component Responsibilities

* data plane - Envoy

  Handles the business requests to the gateway, executes the corresponding gateway policies according to the xDS configuration resources.

* control plane - Istio + HTNN controller

  Provides cloud-native standard gateway resource abstractions, such as the industry-standard k8s Gateway API and Istio CRD, as well as HTNN CRD abstracted for the plugin platform.

  In addition to subscribing to gateway configuration resources from k8s, the control plane also subscribes to resources like Service, Endpoints, and Pod for service discovery, translates these resources into xDS, and pushes them to the Envoy data plane.

* k8s

  The cloud-native collaboration layer, acting as a centralized database, responsible for the collaboration between the console and the control plane controllers, as well as between Istio and other k8s controllers.

* console

  The product layer abstracts product capabilities based on business scenarios, such as domain name access/API access, and change control capabilities. At the same time, as a centralized management, it provides multi-cluster management capabilities.

  The HTTP API provided by the console translates business change requests into cloud-native standard gateway resources and writes them to k8s.

* Dashboard

  It provides a user-friendly operation interface and various visualized metric browsing capabilities.

* Golang Plugin Hub

  A community-built hub containing various general-purpose plugins for easy sharing within the community.

## Highlights

* Fully open source, ready to use

  All components of HTNN are fully open-source, including business management at the product layer, ready to use out of the box.

* Cloud-native Standard

  In addition to supporting industry-standard resources such as k8s Gateway API and Istio CRD, the extended capabilities of HTNN also conform to cloud-native standard specifications and provide abstractions for configuration resources.

* Flexible and Extensible

  Envoy was originally used more for east-west Service Mesh scenarios, focusing more on multi-protocol support and service governance.

  The north-south gateway carries external internet traffic, and relatively speaking, there is a higher demand for traffic control, which in turn places higher demands on the extensibility of the gateway.

* High Development Efficiency

  In addition to the data plane providing a Golang extension mechanism, the console is also implemented in Golang. In addition to these two components that developers often need to extend, the control plane's Istio and HTNN-controller are also implemented in Golang, which means developers can use Golang to handle the full stack.

  Relying on the Golang development ecosystem, we can extend and customize HTNN very efficiently.

* Multi-cluster Management

  In today's highly prevalent cloud-native era, having multiple clusters has become the norm, and HTNN has built-in multi-cluster management mechanisms that can easily manage multiple clusters.

* Efficient Attack Prevention

  Leveraging the efficient processing mechanism of xdp, it can handle larger-scale attacks, and can also achieve 4&7 layer linkage to build an efficient interception mechanism.

## Quick Start

Can't wait? [Welcome to try it out quickly](./quick_start.md)

## FAQ

1. Why choose Envoy

   As a proxy software born in the cloud-native era, Envoy is designed to adapt to the cloud-native environment, with features such as dynamic configuration changes via xDS.

   Envoy also has large-scale deployments in the industry, with millions of Envoy instances at Ant Financial. Its performance and stability are well-recognized.

2. Why choose Istio

   In addition to serving as a north-south ingress gateway, HTNN is also used for east-west Service Mesh. While Istio has been deeply involved in the east-west direction for many years and covers the north-south direction as well, in terms of maturity, Istio is the more suitable choice.

3. The relationship between HTNN and MOSN

   MOSN is a Golang-implemented data plane, which complements the advantages of Envoy through the MoE architecture, forming the data plane of HTNN.

   Based on this inheritance and evolutionary relationship, HTNN is also a new product incubated by the MOSN community.
