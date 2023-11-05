## How to run test

Assumed you are at the root of this project:

1. Run `make build-test-so` to build the Go plugins.
2. Run `make integration-test` to run the tests.
3. The test framework will start Envoy to run the Go plugins. The stdout/stderr of the Envoy can be found in `./test-envoy/$test_name`.

## Port usage

The test framework will use `:9999` for the control plane, `:10000` for the Envoy proxy, and `:10001` for the backend server.
