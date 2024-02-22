---
title: 插件集成测试框架
---

## 如何运行测试

假设您位于此项目的根目录：

1. 运行 `make build-test-so` 构建 Go 插件。
2. 运行 `go test -v ./plugins/tests/integration -run TestPluginXX` 来运行选定的测试。

测试框架将启动 Envoy 来运行 Go 插件。Envoy 的 stdout/stderr 输出内容可以在 `./test-envoy/$test_name` 中找到。

一些测试需要第三方服务。您可以通过在 `./plugins/tests/integration/testdata/services` 下运行 `docker-compose up $service` 来启动它们。

## 端口使用

测试框架将使用：

* `:2023` 用于表示错误的端口
* `:9999` 用于控制平面
* `:10000` 用于数据面
* `:10001` 用于后端服务器和模拟外部服务器
