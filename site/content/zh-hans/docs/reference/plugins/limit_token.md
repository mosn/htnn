---
title: Token 限流插件配置说明
---

## 描述

该插件用于对 LLM 请求进行 token 级别的限流，可按 Header、Param、Cookie、Consumer 或 IP 等维度进行全局或独立计数的限流。支持 Redis 分布式存储和 token 统计预测。

## 属性

|        |      |
|--------|------|
| 类型   | 限流插件 |
| 顺序   | 未指定  |
| 状态   | 实验   |

## 配置

| 名称             | 类型                              | 必填 | 校验条件 | 描述                                                                 |
|------------------|-----------------------------------|------|----------|----------------------------------------------------------------------|
| rejected_code    | integer                           | 否   |          | 当请求被限流时返回的 HTTP 状态码，例如 429。                         |
| rejected_msg     | string                            | 否   |          | 请求被拒绝时返回的消息。                                             |
| rule             | [Rule](#rule)                     | 是   |          | 限流规则配置。                                                        |
| redis            | [RedisConfig](#redisconfig)       | 否   |          | Redis 配置，用于分布式限流。                                         |
| token_stats      | [TokenStatsConfig](#tokenstatsconfig) | 否   |          | 用于统计 Prompt/Completion token 并预测 Completion token。如果在计算 Token 限额时发生错误，请求会默认被拒绝。未来版本将支持配置策略，允许用户自定义在出错时是否拒绝请求。 |
| tokenizer        | string                            | 否   |          | 模型适配器类型，例如 "openai"。                                       |
| extractor_config | [GjsonConfig](#gjsonconfig)       | 是   |          | 用于提取请求/响应内容的路径配置。                                     |
| streaming_enabled| boolean                           | 否   |          | 是否支持流式响应的限流计算。                                         |

### Rule

| 名称        | 类型        | 必填 | 描述                                     |
|-------------|------------|------|------------------------------------------|
| limit_by    | oneof      | 是   | 限流依据，可按 Header、Param、Cookie、Consumer 或 IP 设置，仅可选其中一项。 |
| buckets     | array of Bucket | 是 | 限流桶配置，包括突发流量、生成速率和周期。 |
| keys        | array of string | 否 | 提取的 key 表达式（支持正则），用于细化限流粒度。 |

#### Bucket

| 名称   | 类型   | 描述                        |
|--------|-------|-----------------------------|
| burst  | int32 | 突发流量最大 token 数，类似桶容量 |
| rate   | int32 | 令牌生成速率（每秒 tokens 数） |
| round  | int32 | 令牌桶周期                   |

### RedisConfig

| 名称          | 类型   | 必填 | 描述                                |
|---------------|--------|------|-------------------------------------|
| service_addr  | string | 是   | Redis 服务地址，例如 localhost:6379 |
| username      | string | 否   | Redis 用户名（可选）                |
| password      | string | 否   | Redis 密码（可选）                  |
| timeout       | uint32 | 否   | Redis 超时时间，单位秒               |

### TokenStatsConfig

| 名称              | 类型   | 必填 | 描述                                                     |
|------------------|-------|------|----------------------------------------------------------|
| window_size       | int32 | 否   | 滑动窗口大小（统计样本数量上限），默认 1000              |
| min_samples       | int32 | 否   | 启动预测所需最小样本数，默认 10                           |
| max_ratio         | float | 否   | 默认最大 Prompt/Completion token 比例，默认 4.0           |
| max_tokens_per_req| int32 | 否   | 每次请求允许的最大 token 总数，默认 2000                  |
| exceed_factor     | float | 否   | 超出预测值放宽因子，例如 1.5 表示允许 150%，默认 1.5      |

### GjsonConfig

| 名称                         | 类型   | 必填 | 描述                                              |
|-------------------------------|--------|------|---------------------------------------------------|
| request_content_path          | string | 是   | 从请求体提取内容的 GJSON 路径                     |
| request_model_path            | string | 是   | 从请求体提取模型信息的 GJSON 路径                 |
| response_content_path         | string | 是   | 从非流式响应体提取内容的 GJSON 路径               |
| response_model_path           | string | 是   | 从响应体提取模型信息的 GJSON 路径                 |
| response_completion_tokens_path | string | 否  | 从响应体提取 Completion Token 的路径             |
| response_prompt_tokens_path     | string | 否  | 从响应体提取 Prompt Token 的路径                  |
| stream_response_content_path    | string | 否  | 流式响应每个 chunk 提取内容的 GJSON 路径          |
| stream_response_model_path      | string | 否  | 流式响应每个 chunk 提取模型信息的 GJSON 路径      |

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
  LimitToken:
    config:
      rejected_code: 409
      rejected_msg: "请求被限流"
      rule:
        limit_by_header: "Authorization"
        buckets:
          - burst: 100
            rate: 50
            round: 1
      redis:
        service_addr: "localhost:6379"
      token_stats:
        window_size: 1000
        min_samples: 10
        max_ratio: 4.0
        max_tokens_per_req: 2000
        exceed_factor: 1.5
      tokenizer: "openai"
      extractor_config:
        gjson_config:
          request_content_path: "messages.0.content"
          request_model_path: "messages.0.model"
          response_content_path: "choices.0.message.content"
          response_model_path: "choices.0.message.model"
          stream_response_content_path: "choices.0.delta.content"