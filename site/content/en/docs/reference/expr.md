---
title: Expression
---

HTNN uses CEL (Common Expression Language) in many places as a runtime expression for dynamic execution. For the syntax of CEL, see the [official documentation](https://github.com/google/cel-spec). In order to make CEL useful in the context of network proxy, HTNN provides a set of CEL extension libraries for accessing information about expressions. This document describes the CEL extension library.

## Request

| Name               | Type   | Description                                                  |
|--------------------|--------|--------------------------------------------------------------|
| request.path()     | string | The path of the request, e.g. `/x?a=1`                       |
| request.url_path() | string | The path of the request, without the query string, e.g. `/x` |
| request.host()     | string | The host of the request                                      |
| request.scheme()   | string | The scheme of the request, in lowercase form, e.g. `http`    |
| request.method()   | string | The method of the request, in uppercase form, e.g. `GET`     |
| request.id()       | string | The ID in the `x-request-id` request header                  |