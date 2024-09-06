If possible, please submit the patch to the upstream first.

## Description

This list documents each patch:

* istio
    * 1.21
        * 20240410-htnn-go-mod.patch: Embed HTNN controller into istio. We move the `go.mod`, which may be changed more frequently, to a separate patch.
        * 20240410-embed-htnn-controller-gen-tmpl.patch: Embed HTNN controller into istio (the code generator part).
        * 20240410-embed-htnn-controller-go-code.patch: Embed HTNN controller into istio (the Go code part).
        * 20240508-never-remove-ecds-explicitly.patch: Backport https://github.com/istio/istio/commit/aab0fc6bb0655f5822233458c11605d9ef6b8719 to Istio 1.21. The bug occurred when delta xDS is enabled and using ECDS.
        * 20240510-fix-empty-ecds-with-delta-xds.patch: Backport https://github.com/istio/istio/commit/e91027cf0d5242e677a84e5f6f9dd1924d0175c5 to Istio 1.21. The bug occurred when delta xDS is enabled and using ECDS and pilot-agent.
        * 20240529-fix-routes-overwrite-when-merging-same-host-from-multi-virtualservices.patch: Backport https://github.com/istio/istio/commit/0cb5c33595cdfaea732178a4d70265ac0a762255 to Istio 1.21. The filename in the patch is renamed to match the file in Istio 1.21. The bug occurred sometimes when multiple virtualservices has the same domain.
        * 20240823-server-side-filter.patch: Add server-side filters to filter istio CRD.
        * 20240903-dynamic-configs.patch: Add DynamicConfig CRD.
