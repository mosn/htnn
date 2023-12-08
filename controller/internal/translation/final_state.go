package translation

import (
	"fmt"
	"sort"

	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	"mosn.io/moe/controller/internal/istio"
)

const (
	AnnotationInfo = "htnn.mosn.io/info"
)

func envoyFilterName(route *mergedPolicy) string {
	// We use the NsName as the EnvoyFilter name because the host name may contain invalid characters.
	// This design also make it easier to reference the original CR with the EnvoyFilter.
	// As a result, when a VirtualService or something else has multiple hosts, we hold them in the
	// same EnvoyFilter.
	// The `htnn-h` means the HTNN's HTTPFilterPolicy.
	// The namespace & name may be overlapped, so we use `--` as separator to reduce the chance.
	return fmt.Sprintf("htnn-h-%s--%s", route.NsName.Namespace, route.NsName.Name)
}

// finalState is the end of the translation. We convert the state to EnvoyFilter and write it to k8s.
type FinalState struct {
	EnvoyFilters map[string]*istiov1a3.EnvoyFilter
}

type envoyFilterWrapper struct {
	*istiov1a3.EnvoyFilter
	info *Info
}

func toFinalState(_ *Ctx, state *mergedState) (*FinalState, error) {
	efs := istio.DefaultEnvoyFilters()
	efList := []*envoyFilterWrapper{}
	for _, host := range state.Hosts {
		for routeName, route := range host.Routes {
			ef := istio.GenerateRouteFilter(host.VirtualHost, routeName, route.Config)
			name := envoyFilterName(route)
			ef.SetName(name)

			efList = append(efList, &envoyFilterWrapper{
				EnvoyFilter: ef,
				info:        route.Info,
			})
		}
	}

	// Merge EnvoyFilters with same name. The number of EnvoyFilters is equal to the number of
	// configured VirtualServices.
	efws := map[string]*envoyFilterWrapper{}
	for _, ef := range efList {
		name := ef.GetName()
		if curr, ok := efws[name]; ok {
			curr.Spec.ConfigPatches = append(curr.Spec.ConfigPatches, ef.Spec.ConfigPatches...)
			curr.info.Merge(ef.info)
		} else {
			efws[name] = ef
		}
	}

	for name, ef := range efws {
		ef.SetAnnotations(map[string]string{
			AnnotationInfo: ef.info.String(),
		})
		// Sort here to avoid EnvoyFilter change caused by the order of ConfigPatch.
		// Each ConfigPatch should have a unique (vhost, routeName).
		sort.Slice(ef.Spec.ConfigPatches, func(i, j int) bool {
			a := ef.Spec.ConfigPatches[i]
			b := ef.Spec.ConfigPatches[j]
			aVhost := a.Match.GetRouteConfiguration().GetVhost()
			bVhost := b.Match.GetRouteConfiguration().GetVhost()
			if aVhost.Name != bVhost.Name {
				return aVhost.Name < bVhost.Name
			}
			return aVhost.GetRoute().Name < bVhost.GetRoute().Name
		})
		efs[name] = ef.EnvoyFilter
	}

	return &FinalState{
		EnvoyFilters: efs,
	}, nil
}
