---
title: Plugin Integration Test Framework
---

## How to run test

Assumed you are at the root of this project:

1. Run `make build-test-so` to build the Go plugins.
2. Run `go test -v ./plugins/tests/integration -run TestPluginXX` to run the selected tests.

The test framework will start Envoy to run the Go plugins. The stdout/stderr of the Envoy can be found in `./test-envoy/$test_name`.

Some tests require third-party services. You can start them by running `docker-compose up $service` under `./plugins/tests/integration/testdata/services`.

## Port usage

The test framework will use:

* `:2023` to represent invalid port
* `:9999` for the control plane
* `:10000` for the Envoy proxy
* `:10001` for the backend server and mock external server
