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

We can also start Envoy via binary (also known as binary mode). Using binary mode requires configuring the environment variable `TEST_ENVOY_BINARY_PATH` to point to the path of the Envoy binary file. For example, `TEST_ENVOY_BINARY_PATH=$(which envoy) go test -v ./tests/integration -run TestPluginXX`. Note that the Envoy binary and the Go plugin compiled so files need to be compatible:

* The compilation platforms must be consistent
* The glibc versions must be compatible
* The Envoy API versions used must be consistent (see [HTNN's Envoy multi-version support](./dataplane_support.md))

By default, in binary mode, the testing framework will wait 1 second for Envoy to start. This time can be modified via the environment variable `TEST_ENVOY_WAIT_BINARY_TO_START_TIME`. For example, `TEST_ENVOY_BINARY_MODE_WAIT_TIME=2s TEST_ENVOY_BINARY_PATH=$(which envoy) go test -v ./tests/integration -run TestFilterManagerEncode`.

## Port usage

The test framework will occupy the following ports on the host machine:

* `:9998` for the Envoy's Admin API, which can be modified by the environment variable `TEST_ENVOY_ADMIN_API_PORT`
* `:9999` for the control plane, which can be modified by the environment variable `TEST_ENVOY_CONTROL_PLANE_PORT`
* `:10000` for the Envoy proxy, which can be modified by the environment variable `TEST_ENVOY_DATA_PLANE_PORT`

For example, `TEST_ENVOY_CONTROL_PLANE_PORT=19999 go test -v ./tests/integration -run TestPluginXX` will use `:19999` as the control plane port.

## Debugging Failed Test Cases

The application logs and access logs of Envoy will be output to stdout, and can ultimately be found in `$test_dir/test-envoy/$test_name/stdout`.

If Envoy crashes on startup, it is usually because the ABI used by the Go shared library loaded does not match the Envoy started by the testing framework. In this case, it is necessary to set the `PROXY_IMAGE` environment variable to use the correct version of Envoy.

By default, the testing framework will use the `info` level for application logs. If you want to investigate unexpected behavior from Envoy, it is recommended to lower the log level to `debug`:

```go
dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
    LogLevel:        "debug",
})
```
