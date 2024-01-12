---
title: Limit Req
---

## 说明

`limitReq` 插件限制了每秒对此代理的请求数量。其实现是基于[令牌桶算法](https://en.wikipedia.org/wiki/Token_bucket)。为易于理解，可以把（下文提到的） `average` 和 `period` 看做是桶的补充速率，而 `burst` 则代表桶的容量。

## 属性

|       |         |
|-------|---------|
| Type  | Traffic |
| Order | Traffic |

## 配置

| 名称    | 类型                            | 必选 | 校验规则 | 说明                                                                                                                 |
|---------|---------------------------------|------|----------|----------------------------------------------------------------------------------------------------------------------|
| average | int32                           | 是   | gt: 0    | 阈值，默认单位为每秒请求数计                                                                                             |
| period  | [Duration](../../type#duration) | 否   |          | 速率的时间单位。限制速率定义为 `average / period`。默认为 1 秒，即每秒请求数。                                       |
| burst   | int32                           | 否   | gt: 0    | 允许超出速率的请求数。默认为 1。                                                                                     |

请求数是按客户端 IP 计数的。当请求速率超过 `average / period`，且超出的请求数超过 `burst` 时，我们会计算降低速率至预期水平所需的延迟时间。如果所需延迟时间不大于最大延迟，则请求会被延迟。如果所需延迟大于最大延迟，则请求会以 `429` HTTP 状态码被丢弃。
默认情况下，最大延迟是速率的一半（`1 / 2 * average / period`），如果 `average / period` 小于 1，则为 500 毫秒。

## 用法

假设我们为 `http://localhost:10000/` 提供了如下配置：

```yaml
average: 1 # 限制请求到每秒 1 次
```

第一个请求将获得 `200` 状态码，随后的请求将以 `429` 被丢弃：

```
$ while true; do curl -I http://localhost:10000/ 2>/dev/null | head -1 ; done
HTTP/1.1 200 OK
HTTP/1.1 429 Too Many Requests
HTTP/1.1 429 Too Many Requests
```

如果客户端将其请求速率降低到每秒一个以下，所有请求都不会被丢弃：

```
$ while true; do curl -I http://localhost:10000/ 2>/dev/null | head -1 ; sleep 1; done
HTTP/1.1 200 OK
HTTP/1.1 200 OK
```