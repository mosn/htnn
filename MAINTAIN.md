This doc tracks how to maintain the source code of HTNN.

## Release a new version

To release a new version, please follow the steps below:

* Create a new release branch `release/v${version}` from the main branch.
* Create tag `api/v${version}`, then update the `go.mod` which depend on `mosn.io/htnn/api`.
* Do the same things with `types`, `controller` and `plugins`.
* Remove the `go.work` file.
* Update the version in the `manifests/charts/*/Chart.yaml`.
(TBD)

## Upgrade components

### Upgrade Istio

To upgrade Istio, please follow the steps below:

* Discuss the impact of the upgrade. For example, is there any break change, do we need to upgrade K8S, etc.
* Update the base image used in the integration tests.
* Update the ISTIO_VERSION we define in the `common.mk`.
* Update the link `/envoy/v1.xx.y/configuration/` in the doc to the new Envoy version. And `istio/istio/xxx` to the new Istio version.
* Update the charts' dependency versions used in the `manifests/charts/*/Chart.yaml`.

If this is a minor version upgrade, please follow the additional steps below:

* Sync the `manifests/charts/htnn-controller/*` to the latest istio's istiod chart.
