package ir

type finalState struct {
}

func toFinalState(ctx Ctx, state *mergedState) error {
	// generate all the EnvoyFilter
	// diff with the previous output
	// write the delta to k8s
	return nil
}
