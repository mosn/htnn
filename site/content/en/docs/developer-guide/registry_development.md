---
title: Registry development
---

## How to write a registry

Assume you are at the root of this project.

1. Create a directory under `./controller/registries/`.
2. Think about the configuration and write down it into `./controller/registries/$your_registry/config.proto`. Then run `make gen-proto`. The `proto` file uses [proto-gen-valdate](https://github.com/bufbuild/protoc-gen-validate?tab=readme-ov-file#constraint-rules) to define validation. The configuration fields must be in snake style, like `connect_timeout`. See the [official protobuf style](https://protobuf.dev/programming-guides/style/) for the details.
3. Finish the plugin. You can take `./controller/registries/nacos` as an example.
4. Add the doc in the `site/content/$your_language/docs/reference/registries/$your_registry.md`. You can choose to write doc under Simpilified Chinese or English, depends on which is your prime language. We have [tool](https://github.com/mosn/htnn/tree/main/site#cmdtranslator) to translate it to other languages.
5. Add your registry into `./controller/registries/registries.go`.
6. Add integration test in the `./controller/tests/integration/registries/`.