If possible, please submit the patch to the upstream first.

## Description

This list documents each patch:

* istio/
    * 20240410-htnn-go-mod.patch: Embed HTNN controller into istio. We move the go.mod part, which may be changed more frequent, to a separate patch.
    * 20240410-embed-htnn-controller.patch: Embed HTNN controller into istio.
