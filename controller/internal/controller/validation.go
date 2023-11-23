package controller

import (
	"errors"

	"google.golang.org/protobuf/encoding/protojson"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/pkg/plugins"
)

func validateHTTPFilterPolicy(policy *mosniov1.HTTPFilterPolicy) error {
	// TODO: add webhook
	for name, filter := range policy.Spec.Filters {
		p := plugins.LoadHttpPlugin(name)
		if p == nil {
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
