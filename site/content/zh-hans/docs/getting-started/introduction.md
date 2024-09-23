---
title: 概览简介
---

## HTNN 是什么

HTNN（Hyper Trust-Native Network）是一款基于云原生技术的 L3&L4&L7 Cross-Layer 网络全局解决方案产品。

![HTNN](/images/HTNN.png)

目前开源的只是其中的一部分，L7 南北向接入网关。简单起见，本文先用 HTNN 来代指 HTNN L7 南北向接入网关。

## 为什么要开源 HTNN

HTNN 起源于 MOSN，经过几年的发展融合，沉淀了 MoE（MOSN on Envoy）架构，兼具高性能和高研发效能，既可以享受 Envoy 的高性能底座，以及云原生生态，也可以继承 MOSN 中 Golang 的研发效能，开发者生态。

HTNN 是在蚂蚁内部推动落地的网关产品，依托于开源的 Istio/Envoy，以及 MoE 架构，沉淀了不少企业级能力，插件平台，产品化，多集群，变更管控/审计等，也有不少稳定性，可观测性的提升。

借力开源，反哺开源，开源 HTNN 是希望社区也能享受我们建设的产品能力，也希望社区和我们一起共建，打造先进的网关产品。

## Golang 插件扩展

Golang 作为新一代的语言，具有高性能、简洁易学、并发支持强大、开发生态丰富等优势，在网络、云原生等领域有着广泛的应用。

从 MOSN 开始，我们就享受到了 Golang 的巨大优势，因此我们为 Envoy 也提供了 Golang 扩展能力（已进入 Envoy 官方主干），也沉淀了 MoE 架构的基础。

纵观 Envoy 的众多扩展机制：C++，Lua，Ext-proc，Wasm，Golang。Golang 的开发生态是更有吸引力的，在开源世界里，有着丰富的 Golang 库，我们可以直接用于 HTNN 的插件开发，从而高效的完成各种插件的定制开发。

虽然，Wasm 也可以由 Golang 来生成，不过受限于 Wasm 的成熟度，Golang 生成 Wasm 会有诸多的限制，并且性能也不是很理想。以及 Ext-proc 也可以由 Golang 来完成外部处理服务，不过因此引入的跨进程通讯，处理性能会比较差，并且也会引入额外的服务管理成本。

在网关场景，存在非常多的长尾定制需求，有一个好用 & 靠谱的扩展机制，就显得尤为重要了，因此 HTNN 提供了 Golang 插件扩展平台，可以使用全功能的 Golang 来开发网关插件，包括复用现有 Golang 生态成熟的各种库，从而轻松应对各种流量管控诉求。

## HTNN 的架构

HTNN 是全栈开源的完整产品，从数据面，到控制面，到 console 和 Dashboard（Coming soon），我们都会完整的开源。

![HTNN-architecture](/images/HTNN-architecture.png)

整个架构是从下往上，逐级抽象的体现，越底层通用性越强，往上抽象则更倾向于具体场景。

所以，我们的全栈开源，每一层都是标准的，不会强制绑定，也可以按需取用部分组件。

## 组件分工

* 数据面 - Envoy

  承载网关的业务请求，按照 xDS 配置资源，执行对应的网关策略。

* 控制面 - Istio + HTNN controller

  提供云原生标准的网关资源抽象，如业界标准 k8s Gateway API 和 Istio CRD，以及 HTNN 为插件平台抽象的 HTNN CRD。

  控制面除了从 k8s 订阅网关配置资源，还会订阅 Service，Endpoints，Pod 等资源用于服务发现，将这些资源翻译为 xDS 后，推送给 Envoy 数据面。

* k8s

  云原生协作层，作为集中式的数据库，承担 console 和 控制面 controller 之间，以及 istio 和 其他 k8s controller 之间协作的角色。

* console

  产品层，根据业务场景抽象产品能力，例如域名接入/API 接入，以及变更管控能力等；同时，作为集中式管控，提供多集群的管控能力。

  console 对外提供的 HTTP API，根据业务变更请求，翻译为云原生标准网关资源，写入 k8s。

* Dashboard

  提供友好的操作界面，以及各种可视化的指标浏览。

* Golang Plugin Hub

  社区共建，包含各种通用的插件，方便社区共享。

## 有什么亮点

* 全套开源，开箱即用

  HTNN 的所有组件全套开源，包括在产品层的业务管控，开箱即用。

* 云原生标准

  除了支持 k8s Gateway API、Istio CRD 等，这些业界标准资源，HTNN 扩展的能力，也遵循云原生标准规范，提供了配置资源的抽象。

* 灵活可扩展

  Envoy 原来更多用于东西向 Service Mesh 场景，更多注重的是多协议的支持，以及服务治理。

  南北向网关承载的是外部互联网流量，相对于来说，对流量管控的诉求更多，也就对网关的可扩展性有了更高的要求。

* 高研发效率

  除了数据面提供了 Golang 扩展机制，console 也是用 Golang 来实现。除了这两个开发者经常需要扩展的组件，控制面的 Istio 和 HTNN-controller 也是用 Golang 实现的，也就意味着开发者可以用 Golang 搞定全栈。

  依托于 Golang 的开发生态，我们可以非常高效的扩展定制 HTNN。

* 多集群管理

  在云原生高度普及的今天，拥有多集群已然是常态，HTNN 内置了多集群管理机制，可以轻松管理多套集群。

* 高效防攻击

  利用 xdp 的高效处理机制，应对更大规模的攻击，并且可以 4&7 层联动，构建高效的拦截机制。

## 快速上手

等不及了？[欢迎快速上手体验](./quick_start.md)

## FAQ

1. 为什么选择 Envoy

   Envoy 作为云原生时代诞生的 proxy 基础软件，有着适应于云原生的设计，例如 xDS 动态配置变更。

   并且 Envoy 在业界也有大规模的落地应用，在蚂蚁也有百万量级的 Envoy 实例。性能，稳定性也是有目共睹。

2. 为什么选择 Istio

   HTNN 除了用于南北向的接入网关，也用于东西向 Service Mesh。而 Istio 在东西向深耕多年，南北向也有覆盖，就成熟度而言，Istio 是更合适的选择。

3. HTNN 与 MOSN 的关系

   MOSN 是 Golang 实现的数据面，借助于 MoE 架构，MOSN 与 Envoy 优势互补构建了 HTNN 的数据面。

   基于这样的继承演进关系，HTNN 也是 MOSN 社区孵化出来的新产品。

## 社区交流

### 微信群

<img src="/images/wechat_group.png" height=424 width=270  alt="spacewander_lzx"/>

（如过期请加微信：spacewander_lzx，注明“HTNN”）

### 钉钉群

<img src="/images/dingding_group.png" height=492 width=483  alt="dingding_group"/>
