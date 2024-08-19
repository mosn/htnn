---
title: OIDC
---

## Description

The `OIDC` plugin supports integration with any OpenID Connect Provider (OP) by implementing the [OIDC protocol](https://openid.net/developers/how-connect-works/).

## Attribute

|       |         |
|-------|---------|
| Type  | Authn   |
| Order | Authn   |

## Configuration

| Name                      | Type                            | Required | Validation        | Description                                                                                                                                                                                                                                 |
|---------------------------|---------------------------------|----------|-------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| clientId                  | string                          | True     |                   | The client ID.                                                                                                                                                                                                                              |
| clientSecret              | string                          | True     |                   | The client secret.                                                                                                                                                                                                                          |
| issuer                    | string                          | True     | must be valid URI | The URI of the OIDC Provider, like "https://accounts.google.com".                                                                                                                                                                           |
| redirectUrl               | string                          | True     | must be valid URI | The URL where the user is redirected during OIDC authentication. This URL must meet two criteria: 1. Previously registered with the OIDC Provider. 2. This URL and the user-visited URL must use the same OIDC plugin configuration.        |
| scopes                    | string[]                        | False    |                   | This parameter can request the OIDC Provider to return more information about the authenticated user. For specifics, refer to https://openid.net/specs/openid-connect-core-1_0.html#ScopeClaims and the documentation of the provider used. |
| idTokenHeader             | string                          | False    |                   | The ID Token returned by the OIDC Provider will be passed to the upstream via this header. The default is `X-ID-Token`.                                                                                                                     |
| timeout                   | [Duration](../type.md#duration) | False    | > 0s              | The timeout duration. For example, `10s` indicates a timeout of 10 seconds. The default is 3s.                                                                                                                                              |
| disableAccessTokenRefresh | boolean                         | False    |                   | Whether to disable automatic Access Token refresh.                                                                                                                                                                                          |
| accessTokenRefreshLeeway  | [Duration](../type.md#duration) | False    | >= 0s             | Decides how much earlier a token is considered expired than its actual expiration time when determining the need for refresh. This is used to avoid auto-refresh failures due to client-server time mismatches. The default is 10 seconds.  |

## Usage

In this example, we will demonstrate how to integrate with [hydra](https://github.com/ory/hydra) using the OIDC plugin. HTNN also supports other OP integrations. Different OPs may use different approaches to apply for clientId, clientSecret, and redirectUrl, but there should be little difference beyond that.

Suppose we have the following HTTPRoute attached to `localhost:10000`, and a backend server listening on port `8080`:

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

We are running a set of hydra services, with the address `hydra.service`. The service listens on port 4444 for authentication requests and 4445 for management requests. For specific hydra configurations, refer to https://github.com/mosn/htnn/blob/main/plugins/tests/integration/testdata/services/docker-compose.yml.

Execute the following command to apply for client ID credentials and require permission such as refresh_token:

```shell
hydra create client --response-type code,id_token \
    --grant-type authorization_code,refresh_token -e http://hydra.service:4445 \
    --redirect-uri "http://localhost:10000/callback/oidc" --format json
```

The hydra returns the result as follows: `{"client_id":"5730b1ee-3b0e-4395-b9a2-9e83e8eb1956","client_name":"","client_secret":"Rjqxp0~VdERveFkUxWhfi8mK8-",...}`

Use the returning results to complete the OIDC configuration:

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
    oidc:
      config:
        clientId: 5730b1ee-3b0e-4395-b9a2-9e83e8eb1956
        clientSecret: "Rjqxp0~VdERveFkUxWhfi8mK8-"
        redirectUrl: "http://localhost:10000/callback/oidc"
        issuer: "http://hydra.service:4444"
```

After applying the above configuration, by accessing "http://localhost:10000/" in a browser, the user will be redirected to hydra's login page to complete the OIDC authentication process.
