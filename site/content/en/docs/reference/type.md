---
title: Type
---

This documentation describes common type definitions used across different plugins. Definitions are listed in alphabetical order.

## Duration

A string represents the time duration. The string should end with `s`, which means the number of seconds. For example, `10s` and `0.1s`.

## HeaderValue

A `key` / `value` pair, like `{"key":"Accept-Encoding", "value": "gzip"}`.

## StatusCode

HTTP status code in integer enum.

## StringMatcher

A StringMatcher contains the operation plus `ignore_case` boolean flag. Here is the list of operation:

* exact: must match exactly the string specified here
* prefix: must have the prefix specified here
* suffix: must have the suffix specified here
* regex: must match the regular expression specified here. The syntax is Go's RE2 regex syntax.
* contains: must have the substring specified here

For example,

```json lines
{"exact":"foo", "ignore_case": true}
{"prefix":"pre"}
{"regex":"^Cache"}
```

Note that when the StringMatcher is used to match header, the matching is case-insensitive.
