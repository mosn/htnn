---
title: Plugin Integration Test Framework
---

## How to run test

Assumed you are at the `./plugins` or `./api` directory of this project:

1. Run `make build-test-so` to build the Go plugins.
2. Run `go test -v ./tests/integration -run TestPluginXX` to run the selected tests.

The test framework will start Envoy to run the Go plugins. The stdout/stderr of the Envoy can be found in `$test_dir/test-envoy/$test_name`.
The `$test_dir` is where the test files locate, which is `./tests/integration` in this case.

Some tests require third-party services. You can start them by running `docker compose up $service` under `./tests/integration/testdata/services`.

By default, the test framework starts Envoy using the image `envoyproxy/envoy:contrib-$latest`. You can specify a different image by setting the `PROXY_IMAGE` environment variable. For example, `PROXY_IMAGE=envoyproxy/envoy:contrib-v1.29.4 go test -tags envoy1.29 -v ./tests/integration/ -run TestLimitCountRedis` will use the image `envoyproxy/envoy:contrib-v1.29.4`.

You may have noticed that when executing `go test`, we added `-tags envoy1.29`. This is because there are interface differences across different versions of Envoy. In this case, we specified the label for Envoy version 1.29. See [HTNN's Envoy multi-version support](./dataplane_support.md) for details. Note that the version of Envoy being run, the `-tags` parameter in the `go test` command, and the version of the Envoy interface that is depended upon when running `make build-test-so` should be consistent.

## Port usage

The test framework will use:

* `:2023` to represent invalid port
* `:9999` for the control plane
* `:10000` for the Envoy proxy
* `:10001` for the backend server and mock external server
