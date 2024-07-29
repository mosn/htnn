# HTNN

**Builds**

[![test](https://github.com/mosn/htnn/actions/workflows/test.yml/badge.svg)](https://github.com/mosn/htnn/actions/workflows/test.yml)

**Code quality**

[![coverage](https://codecov.io/gh/mosn/htnn/branch/main/graph/badge.svg)](https://codecov.io/gh/mosn/htnn)
[![go report card](https://goreportcard.com/badge/github.com/mosn/htnn)](https://goreportcard.com/report/github.com/mosn/htnn)

---

HTNN is...

## the hub of the Envoy Go plugins.

First of all, go to the `./plugins` directory.

### To use the Envoy Go plugins

Run `make build-so`. Now the Go plugins are compiled into `libgolang.so`.

To know how to use the Go plugins in a static configuration, you can read [./etc/demo.yaml](./etc/demo.yaml) and run `make run-demo`.

### To develop Envoy Go plugins

To know how to add plugins to this repo, you can read [Plugin Development](./site/content/en/docs/developer-guide/plugin_development.md).

To import HTNN to your project and add your private plugin, you can take a look at the [example](./examples/dev_your_plugin).

## Community

### Chinese

#### 微信群

<img src="https://private-user-images.githubusercontent.com/4161644/353108304-66a6f1f7-4870-4524-883b-566e71478f2c.jpg?jwt=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3MjIyNjMxMTQsIm5iZiI6MTcyMjI2MjgxNCwicGF0aCI6Ii80MTYxNjQ0LzM1MzEwODMwNC02NmE2ZjFmNy00ODcwLTQ1MjQtODgzYi01NjZlNzE0NzhmMmMuanBnP1gtQW16LUFsZ29yaXRobT1BV1M0LUhNQUMtU0hBMjU2JlgtQW16LUNyZWRlbnRpYWw9QUtJQVZDT0RZTFNBNTNQUUs0WkElMkYyMDI0MDcyOSUyRnVzLWVhc3QtMSUyRnMzJTJGYXdzNF9yZXF1ZXN0JlgtQW16LURhdGU9MjAyNDA3MjlUMTQyMDE0WiZYLUFtei1FeHBpcmVzPTMwMCZYLUFtei1TaWduYXR1cmU9ZTc3MjYxYzE0ZjlmNGQ0MWUyZmI5ZjBlZjliZWYzZWNhZTZkZjU1YTI5NGJjMmUyZTJjN2RjNDgwYzRmZjFlOSZYLUFtei1TaWduZWRIZWFkZXJzPWhvc3QmYWN0b3JfaWQ9MCZrZXlfaWQ9MCZyZXBvX2lkPTAifQ.v3ELHuZLKS1COUhN7tnHkKkhNjWXFNvdvkJAG7wksug" height=424 width=270 />

## Thanks

**HTNN** would not be possible without the valuable open-source work of projects in the community. We would like to extend a special thank-you to:

- [Envoy](https://www.envoyproxy.io).
- [Istio](https://istio.io).
