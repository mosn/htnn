package ir

import "mosn.io/moe/controller/internal/envoyfilter"

type finalState struct {
}

func toFinalState(ctx *Ctx, state *mergedState) error {
	envoyfilter.GenerateEnvoyFilters()
	envoyfilter.DiffEnvoyFilters()
	return publishCustomResources(ctx)
}

func publishCustomResources(ctx *Ctx) error {
	// write the delta to k8s
	return nil
}
