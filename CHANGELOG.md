## Table of Contents

- [0.5.0](#050)
- [0.4.1](#041)
- [0.4.0](#040)

## 0.5.0

### Breaking Changes

- upgrade the default Envoy version to 1.32: [#789](https://github.com/mosn/htnn/pull/789)
- allow missing consumer for consumer plugins: [#773](https://github.com/mosn/htnn/pull/773)

### Features

- add sentinel plugin: [#740](https://github.com/mosn/htnn/pull/740)
- support running Go code in synchronous way conditionally:
  - [#800](https://github.com/mosn/htnn/pull/800)
  - [#805](https://github.com/mosn/htnn/pull/805)
  - [#811](https://github.com/mosn/htnn/pull/811)
- support running integration test with envoy bin:
  - [#810](https://github.com/mosn/htnn/pull/810)
  - [#814](https://github.com/mosn/htnn/pull/814)

### Fixes

- fix(cors): allowOriginStringMatch & allowMethods are required [#823](https://github.com/mosn/htnn/pull/823)

## 0.4.1

### Features

- add basic trailer processing: [#762](https://github.com/mosn/htnn/pull/762)
- add routePatch plugin: [#769](https://github.com/mosn/htnn/pull/769)

### Fixes

- support getting headers in OnLog phase by every plugin: [#770](https://github.com/mosn/htnn/pull/770)
- the plugin order sent from controller may be wrong [#774](https://github.com/mosn/htnn/pull/774)

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
