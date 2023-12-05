package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"

	"mosn.io/moe/controller/internal/model"
	"mosn.io/moe/pkg/plugins"
	_ "mosn.io/moe/plugins" // register plugins
)

func ValidateHTTPFilterPolicy(policy *HTTPFilterPolicy) error {
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
		cfg := &model.GoPluginConfig{}
		if err := json.Unmarshal(filter.Raw, cfg); err != nil {
			return err
		}
		data, _ := json.Marshal(cfg.Config)
		conf := p.Config()
		if err := protojson.Unmarshal(data, conf); err != nil {
			return fmt.Errorf("failed to unmarshal for filter %s: %w", name, err)
		}

		if err := conf.Validate(); err != nil {
			return fmt.Errorf("invalid config for filter %s: %w", name, err)
		}
	}
	return nil
}

func ValidateVirtualService(vs *istiov1b1.VirtualService) error {
	// TODO: support delegate VirtualService
	if len(vs.Spec.Hosts) == 0 {
		return errors.New("Delegate VirtualService is not supported")
	}
	return nil
}

func ValidateGateway(gw *istiov1b1.Gateway) error {
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
