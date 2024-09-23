---
title: HTNN's Multi-Version Support for Envoy
---

For users who only need the data plane of HTNN, this document will introduce how to choose HTNN features according to their own situations.

HTNN compiles its Go code into a shared library and loads it into Envoy. So its data plane can be divided into two parts, one is Envoy itself, and the other is the shared library compiled from the Go code developed by HTNN. If users want to use HTNN's data plane code separately, it usually means using HTNN's Go code in combination with their own Envoy.

## Introduction to the Data Plane Code

HTNN's data plane code is located in the `./api` and `./plugins` modules. The introduction and extension of these two modules can be found in [How to Develop HTNN Secondarily](./get_involved.md). Both of these modules can be compiled independently into shared libraries. The difference between them is that `./api` only contains the minimalist implementation of HTNN, providing the necessary interfaces for running Go plugins. While `./plugins` provides a set of official Go plugins based on `./api`. If users want to use the official Go plugins, they can import the `./plugins` module in their own `package main`. This way, when compiling into a shared library, it will include the official Go plugins. For a specific implementation, please refer to https://github.com/mosn/htnn/blob/main/plugins/cmd/libgolang/main.go.

## Choosing the Target Data Plane Version

Since the Envoy Golang filter is still under development, almost every version introduces breaking changes. To address this, HTNN introduces a data plane API version selection mechanism, allowing developers to choose the corresponding HTNN data plane code according to their Envoy version.

By default, the target API version of HTNN's data plane code is the latest officially released Envoy version. At the same time, it supports compiling a shared library that can run on previously released Envoy versions by using build tags. The currently supported versions are as follows:

| Version | build tag                           |
|---------|-------------------------------------|
| 1.29    | envoy1.29                           |
| 1.31    | Latest version, no build tag needed |

For example, to compile a shared library that can run on Envoy 1.29, you need to execute the following command:

```shell
CGO_ENABLED=1 go build -tags so,envoy1.29 --buildmode=c-shared ...
```

If the target is the latest officially released Envoy version, no additional build tag is needed:

```shell
CGO_ENABLED=1 go build -tags so --buildmode=c-shared ...
```

If an interface that only exists in the latest Envoy is executed on an older Envoy, the compatibility layer provided by this suite will execute an virtual interface, output an error log, and return a null value.
