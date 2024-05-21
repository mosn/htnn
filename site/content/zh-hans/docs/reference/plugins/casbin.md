---
title: Casbin
---

## 说明

`casbin` 插件集成了强大且高效的开源访问控制库 [casbin](https://casbin.org/zh/docs/overview/)，支持各种访问控制模型，用于在全局范围内执行授权。

## 属性

|       |       |
| ----- | ----- |
| Type  | Authz |
| Order | Authz |

## 配置

| 名称  | 类型  | 必选 | 校验规则 | 说明 |
| ----- | ----- | ---- | -------- | ---- |
| rule  | Rule  | 是   |          |      |
| token | Token | 是   |          |      |

### Rule

| 名称   | 类型   | 必选 | 校验规则  | 说明                                                                                             |
| ------ | ------ | ---- | --------- | ------------------------------------------------------------------------------------------------- |
| model  | string | 是   | min_len: 1 | Casbin 模型文件的路径，参见 https://casbin.org/zh/docs/model-storage#load-model-from-conf-file        |
| policy | string | 是   | min_len: 1 | Casbin 策略文件的路径，参见 https://casbin.org/zh/docs/policy-storage#loading-policy-from-a-csv-file |

### Token

| 名称   | 类型   | 必选 | 校验规则    | 说明                                                                               |
| ------ | ------ | ---- | ----------- | ---------------------------------------------------------------------------------- |
| source | enum   | 否   | [header]    | 查找令牌的位置，默认为 `header`：从配置的请求头 `name` 中获取令牌                    |
| name   | string | 是   | min_len: 1  | 令牌的名称                                                                         |

## 用法

假设我们定义了如下的模型，称为 `./example.conf`:

```conf
[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = (g(r.sub, p.sub) || keyMatch(r.sub, p.sub)) && keyMatch(r.obj, p.obj) && keyMatch(r.act, p.act)
```

以及如下的策略，称为 `./example.csv`:

```csv
# 注意，act (这里是 GET) 应该是大写
p, *, /, GET
p, admin, *, *
g, alice, admin
```

以上配置允许任何人使用 GET 请求访问主页 (/)。然而，只有拥有管理员权限的用户（alice）可以访问其他页面并使用其他请求方法。

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

让我们应用下面的配置：

```yaml
apiVersion: htnn.mosn.io/v1
kind: HTTPFilterPolicy
metadata:
  name: policy
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: default
  filters:
    casbin:
      config:
        rule:
          # 假设我们已经在 Envoy 的 pod 中挂载了 Casbin 数据
          models: ./example.conf
          policy: ./example.csv
        token:
          source: header
          name: user
```

如果我们向主页发起 GET 请求：

```shell
curl -i http://localhost:10000/ -X GET
HTTP/1.1 200 OK
```

但如果未授权的用户试图访问任何其他页面，他们将收到 403 错误：

```shell
curl -i http://localhost:10000/other -H 'user: bob' -X GET
HTTP/1.1 403 Forbidden
```

只有拥有管理员权限的用户可以访问这些页面：

```shell
curl -i http://localhost:10000/other -H 'user: alice' -X GET
HTTP/1.1 200 OK
```

HTNN 会每 10 秒检查 Casbin 数据文件的更改，并在检测到更改时重新加载。
