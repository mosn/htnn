This doc tracks how to maintain the source code of HTNN.

## Release a new version

To release a new version, please follow the steps below:

* Create a new release branch `release/v${version}` from the main branch. Do the work below on the new branch.
* Create tag `api/v${version}`.
* Commit the changes below (the CI will fail at this point):
    * Update those `go.mod` which depend on `mosn.io/htnn/api`.
    * Remove the `go.work` file.
* Create tag `types/v${version}` for `types` module. Then do the same with `controller` and `plugins`.
* Running `make fmt-go`. Don't panic for "server response: not found" error. The sync of sum.golang.org might take half an hour. Try again later. Commit a new commit after the command succeed. The CI should pass now.
* Create tag `image/v${version}` to trigger image building.
* When the image is ready, update the version in the `manifests/charts/*/Chart.yaml`, submit as a new commit.
* The CI will create a new chart package. The artifacthub will scrape the new package and update the version later.
* Try the quick start guide with the new version. Note that you may need to delete the installed chart before installing the new one,
 as `helm install` will not upgrade the charts.
* Add the `go.work` back, and merge the release branch to the main branch.

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
