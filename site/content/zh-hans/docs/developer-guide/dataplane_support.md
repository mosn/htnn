---
title: HTNN 对 Envoy 的多版本支持
---

对于只需要 HTNN 数据面的用户，本文将介绍如何根据自己的情况选用 HTNN 的功能。

HTNN 会将自己的 Go 代码编译成 shared library，加载到 Envoy 当中。所以它的数据面可以分成两部分，一个是 Envoy 本身，另一个是 HTNN 开发的 Go 代码所编译出的 shared library。用户如果要单独使用 HTNN 的数据面代码，通常是指结合自己的 Envoy 来使用 HTNN 的 Go 代码。

## 数据面代码介绍

HTNN 数据面代码位于 `./api` 和 `./plugins` 模块。这两个模块的介绍以及拓展的方式，可以参考 [如何二次开发 HTNN](./get_involved.md)。这两个模块都能独立编译成 shared library。他们的区别在于 `./api` 只包含了最小化的 HTNN 实现，提供了运行 Go 插件所需的接口。而 `./plugins` 在 `./api` 的基础上提供了一组官方的 Go 插件。如果用户想要使用官方的 Go 插件，可以在自己的 `package main` 里 import `./plugins` 模块。这样在编译成 shared library 时就会包含官方 Go 插件。具体实现可以参考 https://github.com/mosn/htnn/blob/main/plugins/cmd/libgolang/main.go 这个文件。

注意由于 `./plugins` 模块的依赖是通过 `go.work` 管理的，如果用户要 import 没有打过 tag 的 `./plugins` 模块版本，需要手动将它所依赖的 `./api` 和 `./types` 模块版本保持一致。如下所示：

```go.mod
require (
    mosn.io/htnn/api v0.3.3-0.20240910021016-dd32dd2d331f // indirect
    mosn.io/htnn/plugins v0.3.3-0.20240912020652-82b6aa8de677
    mosn.io/htnn/types v0.3.3-0.20240910021016-dd32dd2d331f
)
```

如果是打过 tag 的版本，如 `mosn.io/htnn/plugins v0.3.2`，直接 require 该模块即可。

## 选择目标数据面版本

由于 Envoy Golang filter 尚处于发展阶段，几乎每个版本都会引入 break change。为此 HTNN 引入了一套数据面 API 版本选择机制，开发者能够根据自己的 Envoy 版本选择对应的 HTNN 数据面代码。

默认情况下 HTNN 数据面代码的目标 API 版本为 Envoy 1.32。通过 build tag 可以选择其他受支持 Envoy 版本的 shared library 实现。Envoy 1.39 仅支持外部自管的 Envoy contrib 数据面，不会升级 HTNN 默认 Istio 数据面、Helm chart 或 controller。

| 版本 | build tag                  | 最低 Go 版本 |
|------|----------------------------|--------------|
| dev  | envoydev                   | 1.24.6       |
| 1.39（仅外部） | envoy1.39          | 1.25         |
| 1.38 | envoy1.38                  | 1.24.6       |
| 1.37 | envoy1.37                  | 1.24.6       |
| 1.36 | envoy1.36                  | 1.22         |
| 1.35 | envoy1.35                  | 1.22         |
| 1.32 | 最新版本，不需要 build tag | 1.22         |
| 1.31 | envoy1.31                  | 1.22         |
| 1.29 | envoy1.29                  | 1.22         |

举个例子，编译在 Envoy 1.29 上可运行的 shared library 需要执行下面命令：

修改 `go.mod`，把 Envoy SDK 替换成和 Envoy 一致的版本：

```go.mod
replace github.com/envoyproxy/envoy => github.com/envoyproxy/envoy v1.29.5
```

然后编译：

```shell
CGO_ENABLED=1 go build -tags so,envoy1.29 --buildmode=c-shared ...
```

如果目标是默认 Envoy 1.32，则不需要额外的 build tag：

```shell
CGO_ENABLED=1 go build -tags so --buildmode=c-shared ...
```

如果目标是外部 Envoy 1.39.0 数据面，请使用 Go 1.25，将 Envoy SDK replace 为 `v1.39.0`，并使用 `envoy1.39` tag。builder 的 glibc 必须与运行时镜像兼容，且 shared library 必须为运行时架构构建：

```shell
CGO_ENABLED=1 go build -tags so,envoy1.39 --buildmode=c-shared ...
```

SDK 与 `envoyproxy/envoy:contrib-v1.39.0` 运行时镜像必须使用相同 Envoy patch 版本。加载失败时，请检查 Go 版本、replace 指令、build tag、架构、builder glibc 版本和 Envoy 启动日志。

如果在旧的 Envoy 上执行只有最新 Envoy 才存在的接口，会执行到这套兼容层提供的虚假接口，输出错误日志并返回空值。
