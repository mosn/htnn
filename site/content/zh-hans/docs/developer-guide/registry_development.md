---
title: Registry 开发
---

## 如何开发 registry

假设您位于此项目的根目录。

1. 在 `./types/registries/` 下创建一个目录。目录名为你的 registry 名称。
2. 思考需要什么配置，并将其写入 `./controller/registries/$your_registry/config.proto`。然后运行 `make gen-proto`。`proto` 文件使用 [proto-gen-validate](https://github.com/bufbuild/protoc-gen-validate?tab=readme-ov-file#constraint-rules) 来定义校验规则。配置字段必须使用蛇形风格，如 `connect_timeout`。详细信息请查看 [官方 protobuf 风格指南](https://protobuf.dev/programming-guides/style/)。
3. 将您的 registry 添加到 `./types/registries/registries.go` 中。
4. 在 `./controller/registries/` 下创建一个目录，该目录名称要和步骤一创建的目录一致。
5. 在上面创建的目录下完成 registry 开发。您可以参考 `./controller/registries/nacos` 作为示例。
6. 在 `site/content/$your_language/docs/reference/registries/$your_registry.md` 中添加文档。您可以选择用简体中文或英文编写文档，这取决于您的主要语言。我们有 [工具](https://github.com/mosn/htnn/tree/main/site#cmdtranslator) 可以将其翻译成其他语言。
7. 将您的 registry 添加到 `./controller/registries/registries.go` 中。
8. 在 `./controller/tests/integration/registries/` 中添加集成测试。