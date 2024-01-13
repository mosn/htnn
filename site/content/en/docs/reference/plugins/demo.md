---
title: Demo
---

## Description

The `demo` plugin is used to show how to add a plugin to htnn.

## Attribute

|       |             |
|-------|-------------|
| Type  | General     |
| Order | Unspecified |

## Configuration

| Name     | Type   | Required | Validation | Description                                                                      |
|----------|--------|----------|------------|----------------------------------------------------------------------------------|
| hostName | string | True     | min_len: 1 | The request header name which will contain `hello, ...` greeting to the upstream |

## Usage

By configuring `{"hostName":"John Doe"}`, this plugin will insert a header `John Doe: hello, $guest_name` in the request. The value of `$guest_name` is determined by the value of filter state name `guest_name`.
