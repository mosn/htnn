---
title: 类型
---

本文档描述了不同插件中通用的类型定义。定义按字母顺序排列。

## 说明

## Duration

字符串表示时间持续期。字符串应以 `s` 结尾，表示秒数。例如，`10s` 和 `0.1s`。

## HeaderValue

一个 `key` / `value` 对，如 `{"key":"Accept-Encoding", "value": "gzip"}`。

## StatusCode

HTTP 状态码的整数枚举。

## StringMatcher

StringMatcher 包含操作以及 `ignore_case` 的布尔标志。以下是操作列表：

* exact: 必须完全匹配此处指定的字符串
* prefix: 必须具有此处指定的前缀
* suffix: 必须具有此处指定的后缀
* regex: 必须匹配此处指定的正则表达式。语法是 Go 的 RE2 正则语法。
* contains: 必须含有此处指定的子字符串

例如，

```
{"exact":"foo", "ignore_case": true}
{"prefix":"pre"}
{"regex":"^Cache"}
```

请注意，当 StringMatcher 用于匹配 header，匹配是不区分大小写的。
