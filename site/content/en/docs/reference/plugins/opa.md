---
title: OPA
---

## Description

The `opa` plugin integrates with [Open Policy Agent (OPA)](https://www.openpolicyagent.org).
You can use it to interact with remote OPA service (the remote mode), or authorize the request via local policy code (the local mode).

## Attribute

|        |              |
|--------|--------------|
| Type   | Authz        |
| Order  | Authz        |
| Status | Experimental |

## Configuration

| Name   | Type   | Required | Validation | Description |
|--------|--------|----------|------------|-------------|
| remote | Remote | False    |            |             |
| local  | Local  | False    |            |             |

Either `remote` or `local` is required.

### Remote

| Name    | Type   | Required | Validation        | Description                                               |
|---------|--------|----------|-------------------|-----------------------------------------------------------|
| url     | string | True     | must be valid URI | The url to the OPA service, like `http://127.0.0.1:8181/` |
| policy  | string | True     | min_len: 1        | The name of the OPA policy.                               |
| timeout | [Duration](../type.md#duration) | False    |                  | http client timeout                                       |

### Local

| Name   | Type   | Required | Validation | Description                 |
|--------|--------|----------|------------|-----------------------------|
| text   | string | True     | min_len: 1 | The policy code             |

## Data exchange

Assumed the original client request is:

```shell
GET /?a=1&b= HTTP/1.1
Host: localhost:10000
Pet: dog
Fruit: apple
Fruit: banana
```

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
  },
  "custom_response": {
    "body": "Authentication required. Please provide valid authorization header.",
    "status_code": 401,
    "headers": {
      "WWW-Authenticate": [
        "Bearer realm=\"api\""
      ],
      "Content-Type": [
        "application/json"
      ]
    }
  }
}
```

* `allow` indicates whether the request is allowed.
* `custom_response` contains the optional response details (e.g., message, status code, headers) to be returned instead of the default response.

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
kind: FilterPolicy
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

```shell
curl -i -X GET localhost:10000/echo
HTTP/1.1 200 OK
```

If we try to make a request with a different method, the request will fail:

```shell
curl -i -X POST localhost:10000/echo -d "AA"
HTTP/1.1 403 Forbidden
```

### Interact with Local Policy Rules

We can also configure the policy rules directly. Assumed we provide a configuration to `http://localhost:10000/echo` like:

```yaml
opa:
  config:
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

```shell
curl -i -X GET localhost:10000/echo
HTTP/1.1 200 OK
```

If we try to make a request with a different method, the request will fail:

```shell
curl -i -X POST localhost:10000/echo -d "AA"
HTTP/1.1 403 Forbidden
```


### Use of Custom Response

#### Field Format

* **`body`**
  This field represents the message body sent to the client. **If this field exists but no Content-Type is set in the headers, the plugin will automatically add `Content-Type: text/plain` as the default**.

* **`status_code`**
  HTTP status code. This field supports numeric values.

* **`headers`**
  HTTP response headers. Each header value must be represented as an array of strings.

#### Example

```rego
package test
import input.request
default allow = false
allow {
    request.method == "GET"
    startswith(request.path, "/echo")
}
custom_response = {
    "body": "Authentication required. Please provide valid authorization header.",
    "status_code": 401,
    "headers": {
        "WWW-Authenticate": ["Bearer realm=\"api\""],
        "Content-Type": ["application/json"]
    }
} {
    request.method == "GET"
    startswith(request.path, "/x")
}
```

In this example:

* Requests to `/echo` are allowed.
* Requests to `/x` will be denied with a `401 Unauthorized` status and a JSON-formatted error message, along with appropriate headers.

#### Notes

1. When working with a remote OPA service, `custom_response` should be added as part of the policy decision result. For the expected JSON format returned by OPA, refer to the **Data Exchange** section.

2. If `allow` is `true`, the `custom_response` will be ignored by plugin.

3. If some or all fields under `custom_response` are missing in the response, please ensure that the field names and types conform to the expected format.
