# Moe

[![test](https://github.com/mosn/moe/actions/workflows/test.yml/badge.svg)](https://github.com/mosn/moe/actions/workflows/test.yml)

Moe is...

## the hub of the Envoy Go plugins.

### To use the Envoy Go plugins

Run `make build-so`. Now the Go plugins are compiled into `libgolang.so`.

To know how to use the Go plugins in a static configuration, you can read [./etc/demo.yaml](./etc/demo.yaml) and run `make run-demo`.

### To develop Envoy Go plugins

To know how to add plugins to this repo, you can read [Plugin Development](./site/content/en/docs/developer-guide/plugin_development.md).

To import Moe to your project and add your private plugin, you can take a look at the [example](./examples/dev_your_plugin).

## Thanks

**MOE** would not be possible without the valuable open-source work of projects in the community. We would like to extend a special thank-you to:

- [Envoy](https://www.envoyproxy.io).
- [Istio](https://istio.io).
