package translation

import (
	"fmt"
	"sort"

	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	"mosn.io/moe/controller/internal/istio"
	"mosn.io/moe/controller/internal/model"
)

func nameFromHost(host *model.VirtualHost) string {
	// We use the NsName as the EnvoyFilter name because the host name may contain invalid characters.
	// This design also make it easier to reference the original CR with the EnvoyFilter.
	// As a result, when a VirtualService or something else has multiple hosts, we hold them in the
	// same EnvoyFilter.
	// The namespace & name may be overlapped, so we use `--` as separator to reduce the chance
	return fmt.Sprintf("htnn-h-%s--%s", host.NsName.Namespace, host.NsName.Name)
}

// finalState is the end of the translation. We convert the state to EnvoyFilter and write it to k8s.
type FinalState struct {
	EnvoyFilters map[string]*istiov1a3.EnvoyFilter
}

func toFinalState(ctx *Ctx, state *mergedState) (*FinalState, error) {
	efs := istio.DefaultEnvoyFilters()
	hosts := []*mergedHostPolicy{}
	for _, host := range state.Hosts {
		if host.Policy != nil {
			hosts = append(hosts, host)
		}
	}
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].VirtualHost.Name < hosts[j].VirtualHost.Name
	})
	for _, host := range hosts {
		ef := istio.GenerateHostFilter(host.VirtualHost, host.Policy)
		name := nameFromHost(host.VirtualHost)
		ef.SetName(name)

		if curr, ok := efs[name]; ok {
			curr.Spec.ConfigPatches = append(curr.Spec.ConfigPatches, ef.Spec.ConfigPatches...)
		} else {
			efs[name] = ef
		}
	}
	return &FinalState{
		EnvoyFilters: efs,
	}, nil
}
