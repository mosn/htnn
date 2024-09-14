---
title: Registry development
---

## How to write a registry

Assume you are at the root of this project.

1. Create a directory under `./types/registries/`. The directory name should be your registry name.
2. Think about what configurations are needed, and write them into `./controller/registries/$your_registry/config.proto`. Then run `make gen-proto`. The `proto` file uses [proto-gen-validate](https://github.com/bufbuild/protoc-gen-validate?tab=readme-ov-file#constraint-rules) to define validation rules. Configuration fields must use snake case, such as `connect_timeout`. For more details, please see the [official protobuf style guide](https://protobuf.dev/programming-guides/style/).
3. Add your registry to `./types/registries/registries.go`.
4. Create a directory under `./controller/registries/`, with the name consistent with the directory created in step one.
5. Complete the development of the registry in the directory created above. You can refer to `./controller/registries/nacos` as an example.
6. Add documentation to `site/content/$your_language/docs/reference/registries/$your_registry.md`. You can choose to write the documentation in Simplified Chinese or English, depending on your primary language. We have [tools](https://github.com/mosn/htnn/tree/main/site#cmdtranslator) that can translate it into other languages.
7. Add your registry to `./controller/registries/registries.go`.
8. Add integration tests in `./controller/tests/integration/registries/`.
