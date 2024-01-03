---
title: Ext Auth
---

## Description

This plugin sends an authorization request to an authorization service to check if the client request is authorized or not.

## Attribute

|       |       |
| ----- | ----- |
| Type  | Authz |
| Order | Authz |

## Configuration

| Name         | Type        | Required | Validation | Description |
| ------------ | ----------- | -------- | ---------- | ----------- |
| http_service | HttpService | True     |            |             |

### HttpService

| Name                   | Type                                        | Required | Validation        | Description                                                                                                                                               |
| ---------------------- | ------------------------------------------- | -------- | ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| url                    | string                                      | True     | must be valid URI | The uri to the external service, like `http://ext_auth/prefix`. The path given by the uri will be used as the prefix of the authorization request's path. |
| timeout                | [Duration](../../type#duration)                                    | False    |                   | The timeout duration. For example, `10s` means the timeout is 10 seconds. Default to 0.2s.                                                               |
| authorization_request  | AuthorizationRequest                        | False    |                   |                                                                                                                                                           |
| authorization_response | AuthorizationResponse                       | False    |                   |                                                                                                                                                           |
| status_on_error        | [StatusCode](../../type#statuscode) | False    |                   | Sets the HTTP status that is returned to the client when the authorization server returns an error or cannot be reached. The default status is `401`.     |
| with_request_body      | bool                                        | False    |                   | Buffer the client request body and send it within the authorization request.                                                                              |

### AuthorizationRequest

| Name           | Type                                               | Required | Validation   | Description                                                                                                                                               |
| -------------- | -------------------------------------------------- | -------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| headers_to_add | [HeaderValue[]](../../type#headervalue) | False    | min_items: 1 | Sets a list of headers that will be included in the request to authorization service. Note that client request header of the same key will be overridden. |

### AuthorizationResponse

| Name                     | Type                                                 | Required | Validation   | Description                                                                                                                                                                     |
| ------------------------ | ---------------------------------------------------- | -------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| allowed_upstream_headers | [StringMatcher[]](../../type.md#stringmatcher) | False    | min_items: 1 | When this is set, authorization response headers that have a correspondent match will be added to the original client request. Note that coexistent headers will be overridden. |
| allowed_client_headers   | [StringMatcher[]](../../type#stringmatcher) | False    | min_items: 1 | When this is set, authorization response headers that have a correspondent match will be added to the client's response when the request is rejected.                           |

## Usage

### Sending authorization request

Each client request runs this plugin will trigger an authorization request. The authorization request will have:

* Method from the original client request
* Host from the original client request
* Path from the original client request, after the configured prefix
* Header `Authorization` from the original client request

Assumed we provide a configuration to `http://localhost:10000/` like:

```yaml
http_service:
    url: "http://127.0.0.1:10001/ext_auth"
```

If we make a GET request with a path called `users`:

```shell
curl -i http://localhost:10000/users -X GET -H "foo: bar" -H "Authorization: xxx"
```

The service listening on `10001` will receive an authorization request like:

```
GET /ext_auth/users HTTP/1.1
Host: localhost:10000
User-Agent: Go-http-client/1.1
Authorization: xxx
Accept-Encoding: gzip
```

You can figure out that the request has the same method and adds its own path to the prefix `ext_auth`.

If we make a different request with a body:

```shell
curl -i http://localhost:10000/users -d 'test'
```

The authorization request will be:

```
POST /ext_auth/users HTTP/1.1
Host: localhost:10000
User-Agent: Go-http-client/1.1
Content-Length: 0
Accept-Encoding: gzip
```

If the `headers_to_add` is configured, extra headers will be set to the authorization request.

### Authorization server response

When the server responds with HTTP status 200, the client request is authorized. If the `allowed_upstream_headers` is configured, authorization response headers that have a correspondent match will be set to the original client request.

When the server is unreachable or the status is 5xx, the client request is rejected with status code configured by `status_on_error`.

When the server returns the other HTTP status, the client request is rejected with the status code returned. If the `allowed_upstream_headers` is configured, authorization response headers that have a correspondent match will be set to the client's response.
