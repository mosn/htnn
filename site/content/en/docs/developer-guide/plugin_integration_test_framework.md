---
title: Plugin Integration Test Framework
---

## How to run test

Assumed you are at the `./plugins`:

1. Run `make build-test-so` to build the Go plugins.
2. Run `go test -v ./tests/integration -run TestPluginXX` to run the selected tests.

The test framework will start Envoy to run the Go plugins. The stdout/stderr of the Envoy can be found in `$test_dir/test-envoy/$test_name`.
The `$test_dir` is where the test files locate, which is `./tests/integration` in this case.

Some tests require third-party services. You can start them by running `docker-compose up $service` under `./tests/integration/testdata/services`.

## Port usage

The test framework will use:

* `:2023` to represent invalid port
* `:9999` for the control plane
* `:10000` for the Envoy proxy
* `:10001` for the backend server and mock external server
