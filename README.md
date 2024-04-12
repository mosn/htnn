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

## Thanks

**HTNN** would not be possible without the valuable open-source work of projects in the community. We would like to extend a special thank-you to:

- [Envoy](https://www.envoyproxy.io).
- [Istio](https://istio.io).
