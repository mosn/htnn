---
title: AI Content Security
---

## 说明

设计用于对请求和响应进行内容审核，支持流式响应。

## 属性

|        |              |
|--------|--------------|
| Type   | General      |
| Order  | Unspecified  |
| Status | Experimental |

## 属性

|        |              |
|--------|--------------|
| Type   | General      |
| Order  | Outer        |
| Status | Experimental |

## 配置

| 名称                           | 类型                                                            | 必需 | 验证  | 描述                                          |
|------------------------------|---------------------------------------------------------------|----|-----|---------------------------------------------|
| moderationTimeout            | 整数                                                            | 否  |     | 外部审核服务所有尝试的总超时时间，单位为毫秒。                     |
| streamingEnabled             | 布尔值                                                           | 否  |     | 是否启用流式响应支持。                                 |
| moderationCharLimit          | 整数                                                            | 是  | > 0 | 单次审核请求的字符限制。如果文本超过此限制，将被分块处理。               |
| moderationChunkOverlapLength | 整数                                                            | 否  |     | 分割大型文本进行审核时，文本块之间的重叠字符数。这有助于在各个块之间保持上下文连贯性。 |
| gjsonConfig                  | [GjsonConfig](#gjsonconfig)                                   | 是  |     | 使用 GJSON 路径提取内容的配置。                         |
| aliyunConfig                 | [AliyunConfig](#aliyunconfig)                                 | 否  |     | 使用阿里云内容审核服务的配置。                             |
| localModerationServiceConfig | [LocalModerationServiceConfig](#localmoderationserviceconfig) | 否  |     | 本地审核服务的配置（主要用于测试）。                          |

**注意：** 您必须在顶层提供**一种**提供商配置：`aliyunConfig`或`localModerationServiceConfig`。

### GjsonConfig

`gjsonConfig`对象的配置。

| 名称                        | 类型                              | 必需 | 验证 | 描述                          |
|---------------------------|---------------------------------|----|----|-----------------------------|
| requestContentPath        | 字符串                             | 否  |    | 从请求体中提取需要审核的内容的 GJSON 路径。   |
| responseContentPath       | 字符串                             | 否  |    | 从非流式响应体中提取内容的 GJSON 路径。     |
| streamResponseContentPath | 字符串                             | 否  |    | 从流式响应的每个数据块中提取内容的 GJSON 路径。 |
| headerFields              | [FieldMapping](#fieldmapping)数组 | 否  |    | 从请求头中提取的字段。                 |
| bodyFields                | [FieldMapping](#fieldmapping)数组 | 否  |    | 使用 GJSON 路径从请求体中提取的字段。      |

#### FieldMapping

定义从源字段到目标字段的映射，用于提取会话 ID 等元数据。

| 名称          | 类型  | 必需 | 验证 | 描述                            |
|-------------|-----|----|----|-------------------------------|
| sourceField | 字符串 | 否  |    | 提取值的源字段（例如，头部名称或 GJSON 路径）。   |
| targetField | 字符串 | 否  |    | 用于提取值的目标字段名称（例如，"SessionId"）。 |

### AliyunConfig

`aliyunConfig`对象的配置。

| 名称              | 类型  | 必需 | 验证 | 描述                                                |
|-----------------|-----|----|----|---------------------------------------------------|
| accessKeyId     | 字符串 | 否  |    | 阿里云 API 认证的 AccessKey ID。                         |
| accessKeySecret | 字符串 | 否  |    | 阿里云 API 认证的 AccessKey Secret。                     |
| region          | 字符串 | 否  |    | 阿里云服务区域（例如，"cn-shanghai"）。                        |
| version         | 字符串 | 否  |    | 使用的阿里云 API 版本（例如，"2022-03-02"）。                   |
| useSessionId    | 布尔值 | 否  |    | 是否使用会话 ID 进行多个请求间的上下文审核。                          |
| maxRiskLevel    | 字符串 | 否  |    | 内容达到或超过此级别将被拒绝。有效值包括"none"、"low"、"medium"、"high"。 |
| timeout         | 整数  | 否  |    | 单个外部审核服务请求的超时时间，单位为毫秒。                            |

### LocalModerationServiceConfig

`localModerationServiceConfig`对象的配置（用于本地集成测试）。

| 名称                 | 类型    | 必需 | 验证 | 描述             |
|--------------------|-------|----|----|----------------|
| baseUrl            | 字符串   | 否  |    | 本地服务的基础 URL。   |
| customErrorMessage | 字符串   | 否  |    | 拒绝时返回的自定义错误消息。 |
| unhealthyWords     | 字符串数组 | 否  |    | 被视为不健康的词汇列表。   |

## 用法

本示例演示如何通过 `AI Content Security` 插件对接内容审核服务和 LLM 推理后端。

假设我们已有如下附加到 `localhost:10000` 的 HTTPRoute，并有一个 LLM 推理服务运行在 `localhost:10901`：

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

这里我们以阿里云提供的 `TextModerationPlus` 服务为例进行测试。
假设你已完成所有开通流程，并已获得有效的 `access_key_id` 和 `access_key_secret`。

同时，LLM 推理 API 的格式应符合 OpenAI 的 Chat Completions API 标准。

以下是一个 `FilterPolicy` 示例配置：

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
          access_key_secret: "our accessKeySecret"
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

应用上述配置后，可通过接口测试工具（如 Postman）访问 `http://localhost:10000/v1/chat/completions` ，
发送符合 OpenAI Chat Completions API 格式的请求，即可体验集成后的内容审核能力。
