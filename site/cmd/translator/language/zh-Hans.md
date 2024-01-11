---
title: OPA
---

## 说明

该插件集成了 [Open Policy Agent (OPA)](https://www.openpolicyagent.org)。
您可以用它与远程 OPA 服务交互（远程模式），或通过本地策略代码授权请求（本地模式）。

## 属性

|       |       |
|-------|-------|
| Type  | Authz |
| Order | Authz |

## 配置

| 名称   | 类型   | 必选 | 校验规则 | 说明 |
|--------|--------|------|----------|------|
| remote | Remote | 是   |          |      |
| local  | Local  | 是   |          |      |

`remote` 或 `local` 之一为必填项。

### Remote

| 名称   | 类型   | 必选 | 校验规则          | 说明｜                                      |
|--------|--------|------|-------------------|---------------------------------------------|
| url    | string | 是   | must be valid URI | OPA 服务的 url，如 `http://127.0.0.1:8181/` |
| policy | string | 是   | min_len: 1        | OPA 策略的名称                              |

## 数据交换

下面是 HTNN 发送给 OPA 的 JSON 数据：

```json
{
}
```
