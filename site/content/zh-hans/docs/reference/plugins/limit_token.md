---
title: Token Limit 插件
---

## 说明

`Token Limit` 插件提供针对 LLM 请求的基于 token 的速率限制。它支持全局限制或基于请求头、URL 参数、Cookie、消费者标识或 IP 地址的单值限制。插件支持 Redis 用于分布式存储，以及 token 统计和预测功能。

## 属性

|        |              |
|--------|--------------|
| Type   | Traffic      |
| Order  | Traffic      |
| Status | Experimental |


## 配置

| 名称             | 类型                              | 必填 | 校验规则 | 说明 |
|------------------|-----------------------------------|------|----------|------|
| rejectedCode     | int32                             | 否   |          | 当请求被拒绝时返回的 HTTP 状态码，例如 429。 |
| rejectedMsg      | string                            | 否   |          | 当请求被拒绝时返回的消息。 |
| rule             | [Rule](#rule)                     | 否   |          | 速率限制规则配置。 |
| redis            | [RedisConfig](#redisconfig)       | 否   |          | 分布式速率限制的 Redis 配置。 |
| tokenStats       | [TokenStatsConfig](#tokenstatsconfig) | 否 |          | 用于跟踪 Prompt/Completion token 并预测 completion token 的配置。 |
| tokenizer        | string                            | 否   |          | LLM 适配器类型，例如 "openai"。 |
| gjsonConfig      | [GjsonConfig](#gjsonconfig)       | 是   |          | 配置从请求/响应中提取内容和元数据。 |
| streamingEnabled | boolean                           | 否   |          | 是否对流式响应启用速率限制。 |

### Rule

| 名称              | 类型     | 必填 | 说明 |
|------------------|----------|------|------|
| limitByConsumer  | string   | 否   | 按消费者进行速率限制。 |
| limitByHeader    | string   | 否   | 按请求头进行速率限制。 |
| limitByParam     | string   | 否   | 按 URL 参数进行速率限制。 |
| limitByCookie    | string   | 否   | 按 Cookie 进行速率限制。 |
| limitByPerConsumer | string | 否   | 按单个消费者进行速率限制。 |
| limitByPerHeader | string   | 否   | 按单个请求头进行速率限制。 |
| limitByPerParam  | string   | 否   | 按单个 URL 参数进行速率限制。 |
| limitByPerCookie | string   | 否   | 按单个 Cookie 进行速率限制。 |
| limitByPerIp     | string   | 否   | 按 IP 进行速率限制。 |
| buckets          | Bucket[] | 否   | token 桶配置，包括突发量、生成速率和周期。 |
| keys             | string[] | 否   | 用于细粒度限制的 key 表达式（支持正则）。 |

#### Bucket

| 名称  | 类型   | 必填 | 说明 |
|-------|--------|------|-----|
| burst | int32  | 否   | 突发流量的最大 token 数（桶容量）。 |
| rate  | int32  | 否   | token 生成速率（每秒 token 数）。 |
| round | int32  | 否   | token 桶周期/间隔。 |

### RedisConfig

| 名称        | 类型   | 必填 | 说明 |
|-------------|--------|------|-----|
| serviceAddr | string | 是   | Redis 服务地址，例如 localhost:6379。 |
| username    | string | 否   | Redis 用户名（可选）。 |
| password    | string | 否   | Redis 密码（可选）。 |
| timeout     | uint32 | 否   | Redis 超时时间（秒）。 |

### TokenStatsConfig

| 名称              | 类型   | 必填 | 说明 |
|-------------------|--------|------|-----|
| windowSize        | int32  | 否   | 滑动窗口大小（最大样本数），默认 1000。 |
| minSamples        | int32  | 否   | 开始预测所需的最少样本数，默认 10。 |
| maxRatio          | float  | 否   | Prompt/Completion token 默认最大比例，默认 4.0。 |
| maxTokensPerReq   | int32  | 否   | 每个请求允许的最大 token 数，默认 2000。 |
| exceedFactor      | float  | 否   | 超出预测 token 的容差因子，默认 1.5。 |

### GjsonConfig

| 名称                            | 类型   | 必填 | 说明 |
|-------------------------------|--------|------|-----|
| requestContentPath            | string | 是   | 从请求体中提取内容的 GJSON 路径。 |
| requestModelPath              | string | 是   | 从请求体中提取模型信息的 GJSON 路径。 |
| responseContentPath           | string | 是   | 从非流式响应中提取内容的 GJSON 路径。 |
| responseModelPath             | string | 是   | 从响应中提取模型信息的 GJSON 路径。 |
| responseCompletionTokensPath  | string | 否   | 从响应中提取 completion token 的 GJSON 路径。 |
| responsePromptTokensPath      | string | 否   | 从响应中提取 prompt token 的 GJSON 路径。 |
| streamResponseContentPath     | string | 否   | 从流式响应的每个 chunk 提取内容的 GJSON 路径。 |
| streamResponseModelPath       | string | 否   | 从流式响应的每个 chunk 提取模型信息的 GJSON 路径。 |

## 模型支持与未来计划

> **当前限制：**
>
> - 仅支持部分 OpenAI 模型（`gpt-3.5-turbo-*`、`gpt-4-*`）。
> - 若请求使用其他模型，token 计算会返回错误，并记录警告日志。
> - Token 计算错误时，请求将被默认拒绝。
>
> **未来计划：**
>
> - 支持更多 LLM 模型的 tokenizer。
> - 提供模型配置文件或注册表，让用户自定义 token 计算规则。
> - 增加出错策略配置项，允许用户选择在计算失败时“拒绝”或“放行”请求。
> - 逐步完善 tokenizer 适配层，使其支持自动识别模型类型并加载对应规则。

## 使用示例

假设你希望对 OpenAI Chat Completions API 请求进行限流，可在 FilterPolicy 中这样配置：

```yaml
filters:
  limittoken:
    config:
      rejectedCode: 429
      rejectedMsg: "请求速率受限"
      rule:
        limitByHeader: "Authorization"
        buckets:
          - burst: 100
            rate: 50
            round: 1
      redis:
        serviceAddr: "localhost:6379"
      tokenStats:
        windowSize: 1000
        minSamples: 10
        maxRatio: 4.0
        maxTokensPerReq: 2000
        exceedFactor: 1.5
      tokenizer: "openai"
      gjsonConfig:
        requestContentPath: "messages.0.content"
        requestModelPath: "messages.0.model"
        responseContentPath: "choices.0.message.content"
        responseModelPath: "choices.0.message.model"
        streamResponseContentPath: "choices.0.delta.content"
