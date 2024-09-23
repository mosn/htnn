---
title: OIDC
---

## 说明

`OIDC` 插件通过实现 [OIDC](https://openid.net/developers/how-connect-works/) 协议，支持对接任意 OpenID Connect Provider (OP) 完成对接过程。

## 属性

|       |         |
|-------|---------|
| Type  | Authn   |
| Order | Authn   |

## 配置

| 名称                      | 类型                                        | 必选 | 校验规则          | 说明                                                                                                                                                    |
|---------------------------|---------------------------------------------|------|-------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| clientId                  | string                                      | 是   |                   | 客户端 ID                                                                                                                                               |
| clientSecret              | string                                      | 是   |                   | 客户端 secret                                                                                                                                           |
| issuer                    | string                                      | 是   | must be valid URI | OIDC Provider 的 URI，如“https://accounts.google.com”                                                                                                  |
| redirectUrl               | string                                      | 是   | must be valid URI | OIDC 认证过程中重定向用户的 URL。该 URL 需要满足两个条件：1. 事先已经在 OIDC Provider 中注册。2. 该 URL 和用户访问的 URL 使用同样的 OIDC 插件配置。   |
| scopes                    | string[]                                    | 否   |                   | 该参数可以要求 OIDC Provider 返回经过身份验证的用户的更多信息。具体可以参考 https://openid.net/specs/openid-connect-core-1_0.html#ScopeClaims 和所用的 Provider 自身的文档。 |
| idTokenHeader             | string                                      | 否   |                   | OIDC Provider 返回的 ID Token 将通过该 header 传给上游。默认为 `X-ID-Token`。                                                                            |
| timeout                   | [Duration](../type.md#duration)             | 否   | > 0s              | 超时时长。例如，`10s` 表示超时时间为 10 秒。默认值为 3s。                                                                                              |
| disableAccessTokenRefresh | bool                                        | 否   |                   | 是否禁止自动刷新 Access Token。                                                                                                                        |
| accessTokenRefreshLeeway  | [Duration](../type.md#duration)             | 否   | >= 0s             | 决定判断是否需要刷新过期令牌时，令牌过期的时间比实际过期时间早多少。它用于避免因客户端与服务器时间不匹配而导致自动刷新失败。默认为 10 秒。           |

## 用法

在本示例里，我们将演示如何通过 OIDC 插件对接 [hydra](https://github.com/ory/hydra)。HTNN 也支持对接其他的 OP。不同的 OP 会使用不同的方式来申请 clientId、clientSecret 和 redirectUrl，除此之外应该没有多少差别。

假设我们有下面附加到 `localhost:10000` 的 HTTPRoute，并且有一个后端服务器监听端口 `8080`：

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
        value: /
    backendRefs:
    - name: backend
      port: 8080
```

我们运行了一组 hydra 服务，地址为 `hydra.service`。该服务监听 4444 端口服务认证请求，监听 4445 端口服务管理请求。具体的 hydra 配置参见 https://github.com/mosn/htnn/blob/main/plugins/tests/integration/testdata/services/docker-compose.yml。

执行下面的命令，申请 client ID 等凭证，并启用 refresh_token 等功能：

```shell
hydra create client --response-type code,id_token \
    --grant-type authorization_code,refresh_token -e http://hydra.service:4445 \
    --redirect-uri "http://localhost:10000/callback/oidc" --format json
```

hydra 返回结果如下：`{"client_id":"5730b1ee-3b0e-4395-b9a2-9e83e8eb1956","client_name":"","client_secret":"Rjqxp0~VdERveFkUxWhfi8mK8-",...}`

使用返回结果完成 OIDC 配置：

```yaml
apiVersion: htnn.mosn.io/v1
kind: FilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    oidc:
      config:
        clientId: 5730b1ee-3b0e-4395-b9a2-9e83e8eb1956
        clientSecret: "Rjqxp0~VdERveFkUxWhfi8mK8-"
        redirectUrl: "http://localhost:10000/callback/oidc"
        issuer: "http://hydra.service:4444"
```

在应用上述配置后，在浏览器中访问 "http://localhost:10000/"，用户会被跳转到 hydra 的登录页面完成 OIDC 认证的流程。
