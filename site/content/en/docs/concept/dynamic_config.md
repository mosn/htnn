---
title: Dynamic Configuration
---

In the process of using the API gateway, users often encounter situations where they need to push configurations irrelevant to routing dynamically. For example, when performing load balancing across multiple data centers in the same city, you need to know the data center information. If you put the data center information in the route configuration, once the data center information changes, you need to find all the routes that use this information and update them again. This approach is complex and inefficient. Therefore, we need a dedicated way to update such configurations dynamically. After the configuration is updated, we need to provide corresponding callbacks on the data plane to consume this configuration. For this purpose, we provide the DynamicConfig mechanism to solve this problem.

Before using DynamicConfig, we need to define the behavior to be triggered on the data plane first.

First, we provide a method for users to register callbacks:

```go
type DynamicConfig interface {
	ProtoReflect() protoreflect.Message
	Validate() error
}

type DynamicConfigProvider interface {
	Config() DynamicConfig
}

type DynamicConfigHandler interface {
	DynamicConfigProvider

	OnUpdate(config any) error
}

func RegisterDynamicConfigHandler(name string, c DynamicConfigHandler) {
}
```

DynamicConfig is generated from protobuf, just like plugin configuration, for example:

```proto
message Config {
  string key = 1 [(validate.rules).string = {min_len: 1}];
}
```

The `Config` method of DynamicConfigHandler bridges DynamicConfig and DynamicConfigHandler.

Each time a configuration is pushed, `OnUpdate` will be triggered, and the business logic should be written inside.

Users can register their own implementation of DynamicConfigHandler:

```go
type demo struct {
}

func (d *demo) Config() DynamicConfig {
	return &Config{}
}

func (d *demo) OnUpdate(config any) error {
	c := config.(*Config)
    ...
}

func init() {
	RegisterDynamicConfigHandler("demo", &demo{})
}
```

For the complete implementation, please refer to: https://github.com/mosn/htnn/blob/main/plugins/dynamicconfigs/demo/config.go.

After defining the callback on the data plane, we can push the configuration through the DynamicConfig CRD. The DynamicConfig CRD has two main fields: `config` and `type`:

```yaml
apiVersion: htnn.mosn.io/v1
kind: DynamicConfig
metadata:
  name: test
  namespace: e2e
spec:
  type: demo # The name of the registered DynamicConfigHandler
  config: # Configuration that conforms to the definition of the corresponding DynamicConfigHandler
    key: value
```

 Whenever the `spec` of DynamicConfig `test` changes, all Envoys within the same namespace will receive the new configuration and execute the `OnUpdate` method of `demo`. Currently, when multiple DynamicConfig resources in the same namespace are configured with the same `type`, only the configuration of one of the DynamicConfig resources will be sent to the data plane.

The DynamicConfig resource only takes effect on the data plane within the same namespace. So we can give the same `type` the ability to issue configurations within different namespaces, which is useful in multi-tenancy or grayscale scenarios. Note: Due to the mechanism of EnvoyFilter, if the namespace is the root namespace of istio (e.g. istio-system by default), this resource will take effect for all data planes.

Note: delete the DynamicConfig resource won't trigger the `OnUpdate` method.
