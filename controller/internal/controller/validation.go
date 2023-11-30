package controller

import (
	"errors"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/pkg/plugins"
)

func validateHTTPFilterPolicy(policy *mosniov1.HTTPFilterPolicy) error {
	// TODO: add webhook
	ref := policy.Spec.TargetRef
	if ref.Group != "networking.istio.io" || ref.Kind != "VirtualService" {
		// relax this restriction once we support more
		return errors.New("unsupported targetRef.group or targetRef.kind")
	}
	for name, filter := range policy.Spec.Filters {
		p := plugins.LoadHttpPlugin(name)
		if p == nil {
			// reject unknown filter in CP, ignore unknown filter in DP
			return errors.New("unknown http filter: " + name)
		}
		conf := p.Config()
		if err := protojson.Unmarshal(filter.Raw, conf); err != nil {
			return err
		}

		if err := conf.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateVirtualService(vs *istiov1b1.VirtualService) error {
	// TODO: support delegate VirtualService
	if len(vs.Spec.Hosts) == 0 {
		return errors.New("Delegate VirtualService is not supported")
	}
	return nil
}

func validateGateway(gw *istiov1b1.Gateway) error {
	// TODO: support it
	for _, svr := range gw.Spec.Servers {
		for _, host := range svr.Hosts {
			if strings.ContainsRune(host, '/') {
				return errors.New("Gateway has host with namespace is not supported")
			}
		}
	}
	return nil
}
