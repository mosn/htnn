---
title: AI Content Security
---

## Description

Designed to perform content moderation on both request and response, with support for streaming response.

## Attribute

|        |              |
|--------|--------------|
| Type   | General      |
| Order  | Unspecified  |
| Status | Experimental |

## Configuration

| Name                         | Type                                                          | Required | Validation | Description                                                                                                                                   |
|------------------------------|---------------------------------------------------------------|----------|------------|-----------------------------------------------------------------------------------------------------------------------------------------------|
| moderationTimeout            | integer                                                       | False    |            | Total timeout across all attempts to the external moderation service, in milliseconds.                                                        |
| streamingEnabled             | boolean                                                       | False    |            | Whether to enable support for streaming responses.                                                                                            |
| moderationCharLimit          | integer                                                       | True     | > 0        | The character limit for a single moderation request. If the text exceeds this limit, it will be chunked.                                      |
| moderationChunkOverlapLength | integer                                                       | False    |            | The number of overlapping characters between text chunks when splitting large text for moderation. This helps maintain context across chunks. |
| gjsonConfig                  | [GjsonConfig](#gjsonconfig)                                   | True     |            | Configuration for extracting content using GJSON paths.                                                                                       |
| aliyunConfig                 | [AliyunConfig](#aliyunconfig)                                 | False    |            | Configuration for using Aliyun's content moderation service.                                                                                  |
| localModerationServiceConfig | [LocalModerationServiceConfig](#localmoderationserviceconfig) | False    |            | Configuration for a local moderation service (primarily for testing).                                                                         |

**Note:** You must provide **one** of the provider configurations: either `aliyunConfig` or
`localModerationServiceConfig` at this top level.

### GjsonConfig

Configuration for the `gjsonConfig` object.

| Name                      | Type                                   | Required | Validation | Description                                                                                                                                                                                                |
|---------------------------|----------------------------------------|----------|------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| requestContentPath        | string                                 | False    |            | GJSON path to extract the content to be moderated from the request body.                                                                                                                                   |
| responseContentPath       | string                                 | False    |            | GJSON path to extract content from a non-streaming response body.                                                                                                                                          |
| streamResponseContentPath | string                                 | False    |            | GJSON path to extract content from each chunk of a streaming response.                                                                                                                                     |
| headerFields              | array of [FieldMapping](#fieldmapping) | False    |            | Whether to use a session ID for contextual moderation across multiple requests. If enabled, make sure to configure the `SessionId` as targetField in `headerFields` or `bodyFields` of `extractor(gjson)`. |
| bodyFields                | array of [FieldMapping](#fieldmapping) | False    |            | Fields to extract from the request body using GJSON paths.                                                                                                                                                 |

#### FieldMapping

Defines a mapping from a source field to a target field, used for extracting metadata like session IDs.

| Name        | Type   | Required | Validation | Description                                                                             |
|-------------|--------|----------|------------|-----------------------------------------------------------------------------------------|
| sourceField | string | False    |            | The source field from which to extract the value (e.g., a header name or a GJSON path). |
| targetField | string | False    |            | The target field name to use for the extracted value (e.g., "SessionId").               |

### AliyunConfig

Configuration for the `aliyunConfig` object.

| Name            | Type    | Required | Validation | Description                                                                                                   |
|-----------------|---------|----------|------------|---------------------------------------------------------------------------------------------------------------|
| accessKeyId     | string  | False    |            | The AccessKey ID for Aliyun API authentication.                                                               |
| accessKeySecret | string  | False    |            | The AccessKey Secret for Aliyun API authentication.                                                           |
| region          | string  | False    |            | The Aliyun service region (e.g., "cn-shanghai").                                                              |
| version         | string  | False    |            | The Aliyun API version to use (e.g., "2022-03-02").                                                           |
| useSessionId    | boolean | False    |            | Whether to use a session ID for contextual moderation across multiple requests.                               |
| maxRiskLevel    | string  | False    |            | Content exceeding or equal this level will be rejected. Valid values include "none", "low", "medium", "high". |
| timeout         | integer | False    |            | Timeout for a single request to the external moderation service, in milliseconds.                             |

### LocalModerationServiceConfig

Configuration for the `localModerationServiceConfig` object (used for local integration testing).

| Name               | Type            | Required | Validation | Description                                        |
|--------------------|-----------------|----------|------------|----------------------------------------------------|
| baseUrl            | string          | False    |            | The base URL of the local service.                 |
| customErrorMessage | string          | False    |            | A custom error message to return upon rejection.   |
| unhealthyWords     | array of string | False    |            | A list of words that will be considered unhealthy. |

## Usage

This example demonstrates how to connect content moderation services with LLM inference backends through the `AI Content Security` plugin.

Assume we already have an HTTPRoute attached to `localhost:10000`, with an LLM inference service running on `localhost:10901`:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: default
spec:
  parentRefs:
    - name: default
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /v1/chat/completions
      backendRefs:
        - name: backend
          port: 10901
```

Here we use Alibaba Cloud's `TextModerationPlus` service as an example for testing.
Assume you have completed all activation processes and obtained valid `access_key_id` and `access_key_secret`.

Additionally, the LLM inference API format should comply with OpenAI's Chat Completions API standard.

Here's a sample `FilterPolicy` configuration:

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: default
  filters:
    AIContentSecurity:
      config:
        moderation_timeout: 3000
        streaming_enabled: true
        moderation_char_limit: 2000
        moderation_chunk_overlap_length: 100
        aliyun_config:
          access_key_id: "your accessKeyId"
          access_key_secret: "your accessKeySecret"
          version: "2022-03-02"
          region: "cn-shanghai"
          use_session_id: true
          max_risk_level: "medium"
          timeout: 500
        gjson_config:
          request_content_path: "messages.0.content"
          response_content_path: "choices.0.message.content"
          stream_response_content_path: "choices.0.delta.content"
          header_fields:
            - source_field: "X-Session-ID"
              target_field: "SessionId"
```

After applying the above configuration, you can use API testing tools (such as Postman) to access `http://localhost:10000/v1/chat/completions` and send requests that comply with the OpenAI Chat Completions API format to experience the integrated content moderation capabilities.
