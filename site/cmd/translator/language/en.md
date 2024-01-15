---
title: OPA
---

## Description

This plugin integrates with [Open Policy Agent (OPA)](https://www.openpolicyagent.org).
You can use it to interact with remote OPA service (the remote mode), or authorize the request via local policy code (the local mode).

## Attribute

|       |       |
| ----- | ----- |
| Type  | Authz |
| Order | Authz |

## Configuration

| Name   | Type   | Required | Validation | Description |
|--------|--------|----------|------------|-------------|
| remote | Remote | True     |            |             |
| local  | Local  | True     |            |             |

Either `remote` or `local` is required.

### Remote

| Name   | Type   | Required | Validation        | Description                                               |
|--------|--------|----------|-------------------|-----------------------------------------------------------|
| url    | string | True     | must be valid URI | The url to the OPA service, like `http://127.0.0.1:8181/` |
| policy | string | True     | min_len: 1        | The name of the OPA policy.                               |

## Data exchange

Here is the JSON data HTNN sends to the OPA:

```json
{
}
```
