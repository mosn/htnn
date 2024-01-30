---
title: 服务发现
---

在 k8s 中，istio 通过 `Service` 来对接后端服务。但是并非所有服务都运行在 k8s 里。为了支持 k8s 之外的服务，istio 提供了 [ServiceEntry](https://istio.io/latest/docs/reference/config/networking/service-entry/) 来描述这类服务的信息。

手工管理 `ServiceEntry` 虽然能解决一些简单场景，但并不适合直接用于生产环境上。HTNN 提供了一套对接服务发现系统的框架，通过从服务发现系统当中订阅服务并转换成 `ServiceEntry`，将已接入到服务发现系统的服务引入到 HTNN 的领域当中。该框架名为 `service registy`，用户可以使用现有的 `registry`，也可以像开发插件一样开发自己的 `registry`。

目前 HTNN 已经提供了针对 Nacos V1 的实现，欢迎各位贡献者提交代码，支持更多的服务发现系统。

相关链接：
* [如何开发 registry](../../devloper-guide/registry_development)
* [现有 registry 的文档](../../reference/registries)