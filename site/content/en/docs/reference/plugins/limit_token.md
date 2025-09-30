---
title: Token Limit Plugin Configuration
---

## Description

This plugin provides token-level rate limiting for LLM requests. It supports global or per-value limits based on headers, URL parameters, cookies, consumer identifiers, or IP addresses. The plugin also supports Redis for distributed storage and token statistics/prediction.

## Attributes

|        |              |
|--------|--------------|
| Type   | Rate Limiting Plugin |
| Order  | Unspecified        |
| Status | Stable             |

## Configuration

| Name             | Type                              | Required | Validation | Description                                                                 |
|------------------|-----------------------------------|----------|------------|-----------------------------------------------------------------------------|
| rejected_code    | integer                           | No       |            | HTTP status code returned when a request is rejected, e.g., 429.           |
| rejected_msg     | string                            | No       |            | Message returned when a request is rejected.                                |
| rule             | [Rule](#rule)                     | Yes      |            | Rate limiting rule configuration.                                           |
| redis            | [RedisConfig](#redisconfig)       | No       |            | Redis configuration for distributed rate limiting.                          |
| token_stats      | [TokenStatsConfig](#tokenstatsconfig) | No   |            | Configuration for tracking Prompt/Completion tokens and predicting completion tokens. |
| tokenizer        | string                            | No       |            | Adapter type for the LLM, e.g., "openai".                                   |
| extractor_config | [GjsonConfig](#gjsonconfig)       | Yes      |            | Configuration for extracting content and metadata from requests/responses.  |
| streaming_enabled| boolean                           | No       |            | Enable rate limiting for streaming responses.                               |

### Rule

| Name    | Type         | Required | Description |
|---------|-------------|----------|-------------|
| limit_by| oneof       | Yes      | Rate limiting criteria. Can be based on Header, Param, Cookie, Consumer, or IP (only one may be selected). |
| buckets | array of Bucket | Yes   | Token bucket configuration, including burst, rate, and round. |
| keys    | array of string | No    | Key expressions (supports regex) used for fine-grained limiting. |

#### Bucket

| Name  | Type   | Description |
|-------|--------|-------------|
| burst | int32  | Maximum tokens for burst traffic (bucket capacity). |
| rate  | int32  | Token generation rate (tokens per second). |
| round | int32  | Token bucket interval/round. |

### RedisConfig

| Name         | Type   | Required | Description |
|--------------|--------|----------|-------------|
| service_addr | string | Yes      | Redis service address, e.g., localhost:6379. |
| username     | string | No       | Redis username (optional). |
| password     | string | No       | Redis password (optional). |
| timeout      | uint32 | No       | Redis timeout in seconds. |

### TokenStatsConfig

| Name               | Type   | Required | Description |
|-------------------|--------|----------|-------------|
| window_size        | int32  | No       | Sliding window size (max number of samples), default: 1000. |
| min_samples        | int32  | No       | Minimum samples required to start prediction, default: 10. |
| max_ratio          | float  | No       | Default max ratio of Prompt/Completion tokens, default: 4.0. |
| max_tokens_per_req | int32  | No       | Maximum tokens allowed per request, default: 2000. |
| exceed_factor      | float  | No       | Allowance factor for exceeding predicted tokens, e.g., 1.5 = 150%, default: 1.5. |

### GjsonConfig

| Name                          | Type   | Required | Description |
|-------------------------------|--------|----------|-------------|
| request_content_path           | string | Yes      | GJSON path to extract content from the request body. |
| request_model_path             | string | Yes      | GJSON path to extract model information from the request body. |
| response_content_path          | string | Yes      | GJSON path to extract content from a non-streaming response. |
| response_model_path            | string | Yes      | GJSON path to extract model information from the response. |
| response_completion_tokens_path| string | No       | GJSON path to extract completion tokens from the response. |
| response_prompt_tokens_path    | string | No       | GJSON path to extract prompt tokens from the response. |
| stream_response_content_path   | string | No       | GJSON path to extract content from each chunk of a streaming response. |
| stream_response_model_path     | string | No       | GJSON path to extract model info from each chunk of a streaming response. |


## Example Usage

To apply token-based rate limiting to OpenAI Chat Completions API requests:

```yaml
filters:
  LimitToken:
    config:
      rejected_code: 429
      rejected_msg: "Request rate-limited"
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
