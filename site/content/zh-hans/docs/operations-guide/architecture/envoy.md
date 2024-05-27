---
title: Envoy
---

## 介绍

Envoy 是 HTNN 数据面的主要组件。HTNN 在 Istio 自己的 [Envoy 发行版](https://github.com/istio/proxy) 上增加了包含 HTNN 数据面业务逻辑的 Go shared library，以及少量 backport 的 Envoy 功能。

HTNN 的 Envoy 百分之百兼容 Istio 自己的 Envoy 发行版。
