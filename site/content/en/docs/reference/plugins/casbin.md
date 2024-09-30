---
title: Casbin
---

## Description

The `casbin` plugin embeds the powerful and efficient open-source access control library [casbin](https://casbin.org/docs/overview) that supports various access control models for enforcing authorization across the board.

## Attribute

|        |              |
|--------|--------------|
| Type   | Authz        |
| Order  | Authz        |
| Status | Experimental |

## Configuration

| Name  | Type  | Required | Validation | Description |
| ----- | ----- | -------- | ---------- | ----------- |
| rule  | Rule  | True     |            |             |
| token | Token | True     |            |             |

### Rule

| Name   | Type   | Required | Validation | Description                                                                                               |
| ------ | ------ | -------- | ---------- | --------------------------------------------------------------------------------------------------------- |
| model  | string | True     | min_len: 1 | The path to Casbin model file, see https://casbin.org/docs/model-storage#load-model-from-conf-file        |
| policy | string | True     | min_len: 1 | The path to Casbin policy file, see https://casbin.org/docs/policy-storage#loading-policy-from-a-csv-file |

### Token

| Name   | Type   | Required | Validation | Description                                                                                         |
| ------ | ------ | -------- | ---------- | --------------------------------------------------------------------------------------------------- |
| source | enum   | False    | [header]   | Where to find the token, default to `header`: fetch token from the configured request header `name` |
| name   | string | True     | min_len: 1 | The name of the token                                                                               |

## Usage

Assumed we define the model as follows called `./example.conf`:

```conf
[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = (g(r.sub, p.sub) || keyMatch(r.sub, p.sub)) && keyMatch(r.obj, p.obj) && keyMatch(r.act, p.act)
```

And the policy as follows called `./example.csv`:

```csv
# Note that the act (GET here) should be uppercase
p, *, /, GET
p, admin, *, *
g, alice, admin
```

The above configuration allows anyone to access the homepage (/) using a GET request. However, only users with admin permissions (alice) can access other pages and use other request methods.

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
```

Let's apply the configuration below:

```yaml
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
    casbin:
      config:
        rule:
          # Assumed that we have mounted the Casbin data in the Envoy's pod
          models: ./example.conf
          policy: ./example.csv
        token:
          source: header
          name: user
```

If we make a GET request to the homepage:

```shell
curl -i http://localhost:10000/ -X GET
HTTP/1.1 200 OK
```

But if an unauthorized user tries to access any other page, they will receive a 403 error:

```shell
curl -i http://localhost:10000/other -H 'user: bob' -X GET
HTTP/1.1 403 Forbidden
```

Only users with admin privileges can access the endpoints:

```shell
curl -i http://localhost:10000/other -H 'user: alice' -X GET
HTTP/1.1 200 OK
```

HTNN will check if the casbin data files are changed each 10 seconds, and reload the changed files if detected.
