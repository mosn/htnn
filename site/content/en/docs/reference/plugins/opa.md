---
title: OPA
---

## Description

The `opa` plugin integrates with [Open Policy Agent (OPA)](https://www.openpolicyagent.org).
You can use it to interact with remote OPA service (the remote mode), or authorize the request via local policy code (the local mode).

## Attribute

|       |       |
| ----- | ----- |
| Type  | Authz |
| Order | Authz |

## Configuration

| Name   | Type   | Required | Validation | Description |
|--------|--------|----------|------------|-------------|
| remote | Remote | True     |            |             |
| local  | Local  | True     |            |             |

Either `remote` or `local` is required.

### Remote

| Name   | Type   | Required | Validation        | Description                                               |
|--------|--------|----------|-------------------|-----------------------------------------------------------|
| url    | string | True     | must be valid URI | The url to the OPA service, like `http://127.0.0.1:8181/` |
| policy | string | True     | min_len: 1        | The name of the OPA policy.                               |

### Local

| Name   | Type   | Required | Validation | Description                 |
|--------|--------|----------|------------|-----------------------------|
| text   | string | True     | min_len: 1 | The policy code             |

## Data exchange

Here is the JSON data HTNN sends to the OPA:

```json
{
    "input": {
        "request": {
            "scheme": "http",
            "path": "/",
            "query": {
                "a": "1",
                "b": ""
            },
            "method": "GET",
            "host": "localhost:10000",
            "headers": {
                "fruit": "apple,banana",
                "pet": "dog"
            }
        }
    }
}
```

Note that:

* `method` is always uppercase, while `host`, `headers` and `scheme` are always lowecase.
* `host` will contain the port if the `:authority` header sent by the client has the port.
* Multiple `headers` and `query` in the same name will be concatenated with ','.

The data can be read as `input` document in OPA. It's the same if you use the local mode.

The OPA policy should define a boolean `allow` and use it to indicate if the request is allowed.

Here is the JSON data OPA sends back to HTNN, set by the configured policy:

```json
{
    "result": {
        "allow": true
    }
}
```

* `allow` indicates whether the request is allowed.

## Usage

### Interact with Remote OPA service

First of all, assumed we have the Open Policy Agent run as `opa.service`.

Let's add a policy:

```shell
curl -X PUT 'opa.service:8181/v1/policies/test' \
    -H 'Content-Type: text/plain' \
    -d 'package test

import input.request

default allow = false

# allow GET request only
allow {
    request.method == "GET"
}'
```

Assumed we have the HTTPRoute below attached to `localhost:10000`, and a backend server listening to port `8080`:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: default
spec:
  parentRefs:
  - name: default
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: backend
      port: 8080
---
apiVersion: htnn.mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    opa:
      config:
        remote:
          url: "http://opa.service:8181"
          policy: test
```

As you can see, the policy `test` will be used to evaluate the input that we send to the OPA service.

Now, to test it out:

```
curl -i -X GET localhost:10000/echo
HTTP/1.1 200 OK
```

If we try to make a request with a different method, the request will fail:

```
curl -i -X POST localhost:10000/echo -d "AA"
HTTP/1.1 403 Forbidden
```

### Interact with Local Policy Rules

We can also configure the policy rules directly. Assumed we provide a configuration to `http://localhost:10000/echo` like:

```yaml
opa:
    local:
        text: |
            package test
            import input.request

            default allow = false

            # allow GET request only
            allow {
                request.method == "GET"
            }
```

Now, to test it out:

```
curl -i -X GET localhost:10000/echo
HTTP/1.1 200 OK
```

If we try to make a request with a different method, the request will fail:

```
curl -i -X POST localhost:10000/echo -d "AA"
HTTP/1.1 403 Forbidden
```