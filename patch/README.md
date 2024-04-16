If possible, please submit the patch to the upstream first.

## Description

This list documents each patch:

* istio/
    * 20240410-htnn-go-mod.patch: Embed HTNN controller into istio. We move the `go.mod`, which may be changed more frequently, to a separate patch.
    * 20240410-embed-htnn-controller.patch: Embed HTNN controller into istio.
