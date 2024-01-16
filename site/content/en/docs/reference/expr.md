---
title: Expression
---

HTNN uses CEL (Common Expression Language) in many places as a runtime expression for dynamic execution. For the syntax of CEL, see the [official documentation](https://github.com/google/cel-spec). In order to make CEL useful in the context of network proxy, HTNN provides a set of CEL extension libraries for accessing information about expressions. This document describes the CEL extension library.

## Request

| name                 | parameter type | return type | description                                                  |
|----------------------|----------------|-------------|--------------------------------------------------------------|
| request.path()       |                | string      | The path of the request, e.g. `/x?a=1`                       |
| request.url_path()   |                | string      | The path of the request, without the query string, e.g. `/x` |
| request.host()       |                | string      | The host of the request                                      |
| request.scheme()     |                | string      | The scheme of the request, in lowercase, e.g., `http`        |
| request.method()     |                | string      | The method of the request, in uppercase, e.g. `GET`          |
| request.header(name) | string         | string      | The header of the request                                    |
| request.query_path() |                | string      | The query string in the path of the request, e.g. `a=1`      |
| request.query(name)  | string         | string      | The query string of the request                              |
| request.id()         |                | string      | The ID in the `x-request-id` request header                  |

If there are multiple values corresponding to the name specified by `request.header(name)` or `request.query(name)`, they will be concatenated with `,`. For example, the following request:

```
GET /x?a=1&a=2 HTTP/1.1
x-hdr: a
x-hdr: b
```

`request.header("x-hdr")` returns `a,b`. `request.query("a")` returns `1,2`.