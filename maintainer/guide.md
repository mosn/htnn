This doc tracks how to maintain the source code of HTNN.

## Release a new version

To release a new version, please follow the steps below:

1. Create tag `api/v${version}`.
2. Commit the changes below to the main branch (the CI will fail at this point):
    * Update those `go.mod` which depend on `mosn.io/htnn/$mod`.
   You can refer to https://github.com/mosn/htnn/commit/d651fdf4e14dda6c769c468141d2b151c4c1b80f for an example.
3. Create tag `types/v${version}` for `types` module. Then do the same with `controller` and `plugins`. Rerun the `test` workflow to verify the changes. Don't panic for "server response: not found" error. The sync of sum.golang.org might take half an hour or longer. Try again later.
4. Create tag `image/v${version}` to trigger image building.
5. Submit a new pull request with the changes below (ensure the CI passes):
    * Once the image is ready, update the version in the `manifests/charts/*/Chart.yaml`.
    * Update the `./examples/dev_your_plugin` to use the released version.
    * Run `make fmt-go`.
    * Promote the maturity of plugins that meet the criteria to stable by updating `maintainer/feature_maturity_level.yaml` and plugin documentation.
    * Update the `CHANGELOG.md`.
   You can refer to https://github.com/mosn/htnn/pull/777/files for an example.
6. Create a release branch `release/v${version}` from the main branch, like `release/v0.3.2`. The CI will create a new chart package.

## Upgrade components

### Adapt latest Envoy

To adapt the latest Envoy, please follow the steps below:

1. Read the Envoy Golang filter's changes and see if there are any changes that need to be adapted.
2. Update `patch/switch-envoy-go-version.sh` to support the latest Envoy version.
3. Pull the latest Envoy contrib-dev image, and update the `.github/workflows/test.yml` to use the latest Envoy Go SDK (the `$FULL_ENVOY_VERSION`) and the image.
4. Run `patch/switch-envoy-go-version.sh` to use the latest Envoy Go SDK. Update `api/pkg/filtermanager/api/` and `api/plugins/tests/pkg/envoy/` to make the test pass.

We can refer to https://github.com/mosn/htnn/pull/886/files for an example.

### Upgrade Istio

To upgrade Istio, please follow the steps below:

* Discuss the impact of the upgrade. For example, is there any break change, do we need to upgrade K8S, etc.
* Update the ISTIO_VERSION we define in the `common.mk` and the dataplane image's Dockerfile.
* Update the link `/envoy/v1.xx.y/configuration/` in the doc to the new Envoy version. And `istio/istio/xxx` to the new Istio version.
* Update the charts' dependency versions used in the `manifests/charts/*/Chart.yaml`.

If this is a minor version upgrade, please follow the additional steps below:

* Sync the `manifests/charts/htnn-controller/*` to the latest istio's istiod chart.
