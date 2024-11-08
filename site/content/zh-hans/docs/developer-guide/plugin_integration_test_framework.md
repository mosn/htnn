---
title: 插件集成测试框架
---

## 如何运行测试

假设您位于本项目的 `./plugins` 或 `./api` 目录下：

1. 运行 `make build-test-so` 构建 Go 插件。
2. 运行 `go test -v ./tests/integration -run TestPluginXX` 来运行选定的测试。

测试框架将启动 Envoy 来运行 Go 插件。Envoy 的 stdout/stderr 输出内容可以在 `$test_dir/test-envoy/$test_name` 中找到。
`$test_dir` 是测试文件所在的目录，在此处指 `./tests/integration`。

一些测试需要第三方服务。您可以通过在 `./tests/integration/testdata/services` 下运行 `docker compose up $service` 来启动它们。

默认情况下，测试框架通过镜像 `envoyproxy/envoy:contrib-$latest` 启动 Envoy。你可以通过设置环境变量 `PROXY_IMAGE` 来指定其他镜像。例如，`PROXY_IMAGE=envoyproxy/envoy:contrib-v1.29.4 go test -tags envoy1.29 -v ./tests/integration/ -run TestLimitCountRedis` 将使用 `envoyproxy/envoy:contrib-v1.29.4` 镜像。

您可能已经注意到，在执行 `go test` 时，我们添加了 `-tags envoy1.29`。这是因为不同版本 Envoy 接口存在差异。在这种情况下，我们指定了 Envoy 1.29 版本的标签。具体见 [HTNN 的 Envoy 多版本支持](./dataplane_support.md)。注意运行的 Envoy 版本，以及 `go test` 命令中的 `-tags` 参数，和 `make build-test-so` 时依赖的 Envoy 接口版本应该保持一致。

## 端口使用

测试框架将占用 host 上的下述端口：

* `:9998` 用于 Envoy 管理 API，可通过环境变量 `TEST_ENVOY_ADMIN_API_PORT` 修改
* `:9999` 用于控制平面，可通过环境变量 `TEST_ENVOY_CONTROL_PLANE_PORT` 修改
* `:10000` 用于数据面，可通过环境变量 `TEST_ENVOY_DATA_PLANE_PORT` 修改

例如，`TEST_ENVOY_CONTROL_PLANE_PORT=19999 go test -v ./tests/integration -run TestPluginXX` 将使用 `:19999` 端口作为控制平面端口。

## 调试失败的测试用例

Envoy 的应用日志和访问日志都会输出到 stdout，最终被写入到 `$test_dir/test-envoy/$test_name/stdout` 中找到。

如果出现 Envoy 在启动时崩溃，通常是因为加载到 Go shared library 使用的 ABI 和测试框架启动的 Envoy 不一样。这种情况下需要通过设置 `PROXY_IMAGE` 环境变量来使用正确的 Envoy 版本。

默认情况下测试框架会使用 `info` 级别的应用日志。如果想要调查和预期不一样的 Envoy 行为，推荐把日志等级降到 `debug`：

```go
dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
    LogLevel:        "debug",
})
```
