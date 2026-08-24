---
title: HTNN's Multi-Version Support for Envoy
---

For users who only need the data plane of HTNN, this document will introduce how to choose HTNN features according to their own situations.

HTNN compiles its Go code into a shared library and loads it into Envoy. So its data plane can be divided into two parts, one is Envoy itself, and the other is the shared library compiled from the Go code developed by HTNN. If users want to use HTNN's data plane code separately, it usually means using HTNN's Go code in combination with their own Envoy.

## Introduction to the Data Plane Code

HTNN's data plane code is located in the `./api` and `./plugins` modules. The introduction and extension of these two modules can be found in [How to develop HTNN to fit your purpose](./get_involved.md). Both of these modules can be compiled independently into shared libraries. The difference between them is that `./api` only contains the minimalist implementation of HTNN, providing the necessary interfaces for running Go plugins. While `./plugins` provides a set of official Go plugins based on `./api`. If users want to use the official Go plugins, they can import the `./plugins` module in their own `package main`. This way, when compiling into a shared library, it will include the official Go plugins. For a specific implementation, please refer to https://github.com/mosn/htnn/blob/main/plugins/cmd/libgolang/main.go.

Note that due to the dependencies of the `./plugins` module being managed by `go.work`, if a user wants to import a version of the `./plugins` module that has not been tagged, they need to manually keep the versions of the `./api` and `./types` modules it depends on consistent. As shown below:

```go.mod
require (
    mosn.io/htnn/api v0.3.3-0.20240910021016-dd32dd2d331f // indirect
    mosn.io/htnn/plugins v0.3.3-0.20240912020652-82b6aa8de677
    mosn.io/htnn/types v0.3.3-0.20240910021016-dd32dd2d331f
)
```

If it's a tagged version, such as `mosn.io/htnn/plugins v0.3.2`, you can directly require that module.

## Choosing the Target Data Plane Version

Since the Envoy Golang filter is still under development, almost every version introduces breaking changes. To address this, HTNN introduces a data plane API version selection mechanism, allowing developers to choose the corresponding HTNN data plane code according to their Envoy version.

By default, HTNN data plane code targets Envoy 1.32. Build tags select a shared library implementation for another supported Envoy version. Envoy 1.39 is supported only for externally managed Envoy contrib data planes; it does not upgrade HTNN's default Istio data plane, Helm charts, or controller.

| Version   | build tag                           | Min Go version |
|-----------|-------------------------------------|----------------|
| dev       | envoydev                            | 1.24.6         |
| 1.39 (external only) | envoy1.39                 | 1.25           |
| 1.38      | envoy1.38                           | 1.24.6         |
| 1.37      | envoy1.37                           | 1.24.6         |
| 1.36      | envoy1.36                           | 1.22           |
| 1.35      | envoy1.35                           | 1.22           |
| 1.32      | Latest version, no build tag needed | 1.22           |
| 1.31      | envoy1.31                           | 1.22           |
| 1.29      | envoy1.29                           | 1.22           |

For example, to compile a shared library that can run on Envoy 1.29, you need to execute the following commands:

Modify `go.mod` and replace the Envoy SDK with the version consistent with Envoy:

```go.mod
replace github.com/envoyproxy/envoy => github.com/envoyproxy/envoy v1.29.5
```

Then compile:

```shell
CGO_ENABLED=1 go build -tags so,envoy1.29 --buildmode=c-shared ...
```

For the default Envoy 1.32 target, no additional build tag is needed:

```shell
CGO_ENABLED=1 go build -tags so --buildmode=c-shared ...
```

To build for an external Envoy 1.39.0 data plane, use Go 1.25, replace the Envoy SDK with `v1.39.0`, and use the `envoy1.39` tag. The builder must be glibc-compatible with the runtime image, and the shared library must be built for the runtime architecture:

```shell
CGO_ENABLED=1 go build -tags so,envoy1.39 --buildmode=c-shared ...
```

Use the same Envoy patch version for the SDK and `envoyproxy/envoy:contrib-v1.39.0` runtime image. If loading fails, check the Go version, replace directive, build tag, architecture, builder glibc version, and Envoy startup logs.

If an interface that only exists in the latest Envoy is executed on an older Envoy, the compatibility layer provided by this suite will execute an virtual interface, output an error log, and return a null value.
