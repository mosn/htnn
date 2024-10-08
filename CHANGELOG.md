## Table of Contents

- [0.4.0](#040)

## 0.4.0

### Breaking Changes

- don't respect the request's Content-Type request header when generating response from Envoy directly, after receiving the response from the upstream: [#744](https://github.com/mosn/htnn/pull/744)

### Features

- add server side filter: [#694](https://github.com/mosn/htnn/pull/694)
- support Consul as ServiceRegistry: [#695](https://github.com/mosn/htnn/pull/695)
- support Nacos v2 as ServiceRegistry:
  - [#703](https://github.com/mosn/htnn/pull/703)
  - [#718](https://github.com/mosn/htnn/pull/718)
- allow setting listener access log via listenerPatch plugin: [#701](https://github.com/mosn/htnn/pull/701)
- add DynamicConfig to dispatch route-independent configuration:
  - [#707](https://github.com/mosn/htnn/pull/707)
  - [#713](https://github.com/mosn/htnn/pull/713)
  - [#716](https://github.com/mosn/htnn/pull/716)
- adapt the latest Envoy:
  - [#732](https://github.com/mosn/htnn/pull/732)
  - [#742](https://github.com/mosn/htnn/pull/742)
- enable dual stack by default: [#741](https://github.com/mosn/htnn/pull/741)

### Fixes

- avoid unnecessary xDS generation for our CRD: [#719](https://github.com/mosn/htnn/pull/719)
