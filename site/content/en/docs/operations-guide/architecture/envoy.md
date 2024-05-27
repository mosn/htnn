---
title: Envoy
---

## Introduction

Envoy is the primary component of the HTNN data plane. HTNN has added a Go shared library that contains the HTNN data plane business logic, as well as a small number of backported Envoy patches, to Istio's own [Envoy distribution](https://github.com/istio/proxy).

HTNN's Envoy is 100% compatible with Istio's own Envoy distribution.
