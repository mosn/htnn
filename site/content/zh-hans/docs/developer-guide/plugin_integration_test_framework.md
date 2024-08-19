---
title: 插件集成测试框架
---

## 如何运行测试

假设您位于 `./plugins`：

1. 运行 `make build-test-so` 构建 Go 插件。
2. 运行 `go test -v ./tests/integration -run TestPluginXX` 来运行选定的测试。

测试框架将启动 Envoy 来运行 Go 插件。Envoy 的 stdout/stderr 输出内容可以在 `$test_dir/test-envoy/$test_name` 中找到。
`$test_dir` 是测试文件所在的目录，在此处指 `./tests/integration`。

一些测试需要第三方服务。您可以通过在 `./tests/integration/testdata/services` 下运行 `docker compose up $service` 来启动它们。

默认情况下，测试框架通过镜像 `envoyproxy/envoy` 启动 Envoy。你可以通过设置环境变量 `PROXY_IMAGE` 来指定其他镜像。例如，`PROXY_IMAGE=envoyproxy/envoy:contrib-v1.29.4 go test -v ./tests/integration/ -run TestLimitCountRedis` 将使用 `envoyproxy/envoy:contrib-v1.29.4` 镜像。

## 端口使用

测试框架将使用：

* `:2023` 用于表示错误的端口
* `:9999` 用于控制平面
* `:10000` 用于数据面
* `:10001` 用于后端服务器和模拟外部服务器
