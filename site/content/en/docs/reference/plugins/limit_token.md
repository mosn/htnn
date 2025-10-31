---
title: Token Limit Plugin Configuration
---

## Description

This plugin provides token-level rate limiting for LLM requests. It supports global or per-value limits based on headers, URL parameters, cookies, consumer identifiers, or IP addresses. Redis is supported for distributed storage and token statistics/prediction.

## Attribute

|        |              |
|--------|--------------|
| Type   | Traffic      |
| Order  | Traffic      |
| Status | Experimental |

## Configuration

| Name             | Type                              | Required | Description                                                                 |
|------------------|-----------------------------------|----------|-----------------------------------------------------------------------------|
| rejectedCode     | int32                             | False    | HTTP status code returned when a request is rejected, e.g., 429.           |
| rejectedMsg      | string                            | False    | Message returned when a request is rejected.                                |
| rule             | [Rule](#rule)                     | False    | Rate limiting rule configuration.                                           |
| redis            | [RedisConfig](#redisconfig)       | False    | Redis configuration for distributed rate limiting.                          |
| tokenStats       | [TokenStatsConfig](#tokenstatsconfig) | False | Configuration for tracking Prompt/Completion tokens and predicting completion tokens. |
| tokenizer        | string                            | False    | Adapter type for the LLM, e.g., "openai".                                   |
| gjsonConfig      | [GjsonConfig](#gjsonconfig)       | True     | Configuration for extracting content and metadata from requests/responses.  |
| streamingEnabled | boolean                           | False    | Enable rate limiting for streaming responses.                               |

### Rule

| Name             | Type     | Required | Description |
|------------------|----------|----------|-------------|
| limitByConsumer  | string   | False    | Rate limiting by consumer. |
| limitByHeader    | string   | False    | Rate limiting by request header. |
| limitByParam     | string   | False    | Rate limiting by URL parameter. |
| limitByCookie    | string   | False    | Rate limiting by cookie. |
| limitByPerConsumer | string | False    | Rate limiting per consumer. |
| limitByPerHeader | string   | False    | Rate limiting per header. |
| limitByPerParam  | string   | False    | Rate limiting per URL parameter. |
| limitByPerCookie | string   | False    | Rate limiting per cookie. |
| limitByPerIp     | string   | False    | Rate limiting per IP. |
| buckets          | Bucket[] | False    | Token bucket configuration, including burst, rate, and round. |
| keys             | string[] | False    | Key expressions (supports regex) used for fine-grained limiting. |

#### Bucket

| Name  | Type   | Required | Description |
|-------|--------|----------|-------------|
| burst | int32  | False    | Maximum tokens for burst traffic (bucket capacity). |
| rate  | int32  | False    | Token generation rate (tokens per second). |
| round | int32  | False    | Token bucket interval/round. |

### RedisConfig

| Name        | Type   | Required | Description |
|-------------|--------|----------|-------------|
| serviceAddr | string | True     | Redis service address, e.g., localhost:6379. |
| username    | string | False    | Redis username (optional). |
| password    | string | False    | Redis password (optional). |
| timeout     | uint32 | False    | Redis timeout in seconds. |

### TokenStatsConfig

| Name              | Type   | Required | Description |
|-------------------|--------|----------|-------------|
| windowSize        | int32  | False    | Sliding window size (max number of samples), default: 1000. |
| minSamples        | int32  | False    | Minimum samples required to start prediction, default: 10. |
| maxRatio          | float  | False    | Default max ratio of Prompt/Completion tokens, default: 4.0. |
| maxTokensPerReq   | int32  | False    | Maximum tokens allowed per request, default: 2000. |
| exceedFactor      | float  | False    | Allowance factor for exceeding predicted tokens, default: 1.5. |

### GjsonConfig

| Name                          | Type   | Required | Description |
|-------------------------------|--------|----------|-------------|
| requestContentPath            | string | True     | GJSON path to extract content from the request body. |
| requestModelPath              | string | True     | GJSON path to extract model information from the request body. |
| responseContentPath           | string | True     | GJSON path to extract content from a non-streaming response. |
| responseModelPath             | string | True     | GJSON path to extract model information from the response. |
| responseCompletionTokensPath  | string | False    | GJSON path to extract completion tokens from the response. |
| responsePromptTokensPath      | string | False    | GJSON path to extract prompt tokens from the response. |
| streamResponseContentPath     | string | False    | GJSON path to extract content from each chunk of a streaming response. |
| streamResponseModelPath       | string | False    | GJSON path to extract model info from each chunk of a streaming response. |

## Model Support and Future Plans

> **Current Limitations:**
>
> - Only partial support for OpenAI models (`gpt-3.5-turbo-*`, `gpt-4-*`).
> - If other models are used, token calculation will return an error and a warning log will be recorded.
> - When a token calculation error occurs, the request will be **rejected by default**.
>
> **Future Plans:**
>
> - Support additional LLM model tokenizers.
> - Provide model configuration files or registries that allow users to define custom token calculation rules.
> - Add configurable error-handling policies, allowing users to choose whether to *reject* or *allow* requests when calculation fails.
> - Gradually improve the tokenizer adaptation layer to automatically detect model types and load the appropriate rules.

## Example Usage

To apply token-based rate limiting to OpenAI Chat Completions API requests:

```yaml
filters:
  limittoken:
    config:
      rejectedCode: 429
      rejectedMsg: "Request rate-limited"
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
