This doc tracks how to maintain the source code of HTNN.

## Upgrade components

### Upgrade Istio

To upgrade Istio, please follow the steps below:

1. Discuss the impact of the upgrade. For example, is there any break change, do we need to upgrade K8S, etc.
2. Update the base image used in the integration / e2e tests.
3. Update the ISTIO_VERSION we define in the Makefile.
4. Update the versions of istio, envoy and go-control-plane package in the `go.mod` and `go.sum`.
5. Update the link `/envoy/v1.28.0/configuration/` in the doc to the new Envoy version.
