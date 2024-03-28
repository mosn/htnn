This doc tracks how to maintain the source code of HTNN.

## Release a new version

To release a new version, please follow the steps below:

1. Create a new release branch `release/v${version}` from the main branch.
2. Update the `VERSION` file at the root of this repository.
3. Create tag `api/v${version}`, then update the `go.mod` which depend on `mosn.io/htnn/api`.
4. Remove the `go.work` file.
(TBD)

## Upgrade components

### Upgrade Istio

To upgrade Istio, please follow the steps below:

1. Discuss the impact of the upgrade. For example, is there any break change, do we need to upgrade K8S, etc.
2. Update the base image used in the integration / e2e tests.
3. Update the ISTIO_VERSION we define in the Makefile.
4. Update the versions of istio, envoy and go-control-plane package in the `go.mod` and `go.sum`.
5. Update the link `/envoy/v1.xx.y/configuration/` in the doc to the new Envoy version.
