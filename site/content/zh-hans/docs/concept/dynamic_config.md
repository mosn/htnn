---
title: 动态配置
---

在 API 网关的使用过程中，用户经常会遇到需要动态下发和路由无关的配置的情况。比如做同城多机房负载均衡，需要知道机房信息。如果在路由配置里放机房信息，一旦机房信息发生变化，需要找出所有用到该信息的路由，并重新下发。这么做，逻辑复杂，而且效率低。所以我们需要有一种专门的方式，单独动态下发这一类配置。配置下发之后，我们需要在数据面上提供对应的回调来消费这种配置。为此，我们提供了 DynamicConfig 机制来解决这个问题。

在使用 DynamicConfig 之前，我们需要先在数据面上定义需要触发的行为。

首先，我们提供让用户注册回调的方法：

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

DynamicConfig 和插件配置一样，都是通过 protobuf 生成的，比如：

```proto
message Config {
  string key = 1 [(validate.rules).string = {min_len: 1}];
}
```

DynamicConfigHandler 的 `Config` 的方法桥接了 DynamicConfig 和 DynamicConfigHandler。
每次推送配置都会触发 `OnUpdate`，具体业务逻辑写在里面。

用户注册自己实现的 DynamicConfigHandler：

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

完整的实现参见：https://github.com/mosn/htnn/blob/main/plugins/dynamicconfigs/demo/config.go。

在数据面上定义了回调之后，我们就可以通过 DynamicConfig CRD 来下发配置了。DynamicConfig CRD 里面有 config 和 type 两个主要的字段：

```yaml
apiVersion: htnn.mosn.io/v1
kind: DynamicConfig
metadata:
  name: test
  namespace: e2e
spec:
  type: demo # 注册的 DynamicConfigHandler 名称
  config: # 符合对应 DynamicConfigHandler 配置定义的配置
    key: value
```

每次 DynamicConfig test 的 `spec` 变化时，同一个 namespace 内的所有 Envoy 上都会收到新的配置，并执行 `demo` 的 OnUpdate 方法。目前当多个同一个 namespace 的 DynamicConfig 资源配置同一个 `type` 时，只有其中一个 DynamicConfig 的配置会发给数据面。

DynamicConfig 资源只对同一个 namespace 内的数据面生效。所以我们可以给同一个 `type` 下发不同 namespace 内的配置，这在多租或灰度场景下很有用。注意：由于 EnvoyFilter 的机制，如果 namespace 是 istio 的 root namespace（比如默认的 istio-system），该资源将对所有数据面生效。

注意：删除 DynamicConfig 资源不会触发 `OnUpdate` 方法。
