---
title: Debug Mode
---

## 说明

`debugMode` 插件用于在目标路由上开启调试模式。

## 属性

|       |         |
|-------|---------|
| Type  | General |
| Order | Access  |

## 配置

| 名称    | 类型    | 必选 | 校验规则 | 说明             |
|---------|---------|------|----------|------------------|
| slowLog | SlowLog | 否   |          | 慢日志相关的配置 |

### SlowLog

| 名称      | 类型                            | 必选 | 校验规则 | 说明                                       |
|-----------|---------------------------------|------|----------|--------------------------------------------|
| threshold | [Duration](../../type#duration) | 是   | > 0s     | 超过该时间则打印错误日志一条，格式见下文。 |

## 用法

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

### 慢日志

让我们应用下面的配置：

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
    debugMode:
      config:
        slowLog:
          threshold: "1s"
```

当请求路由 default 的时间超过 1 秒时，将会打印如下错误日志：

```
[2024-06-14 03:31:38.868][30][error][golang] [contrib/golang/common/log/cgo.cc:24] slow log report: {"total_seconds":4.364525,"request":{"headers":{":autho
rity":["localhost:10000"],":method":["HEAD"],":path":["/echo"],":scheme":["http"],"user-agent":["Go-http-client/1.1"],"x-forwarded-proto":["http"],"x-request-id":["cb212874-58af-469c-b5a3-3bd0c70cb776"]}},"response":{"headers":{":status":["200"],"date":["Fri, 14 Jun 2024 03:31:38 GMT"],"server":["envoy"],"transfer-encoding":["chunked"],"x-envoy-upstream-service-time":["0"]}},"stream_info":{"downstream_remote_address":"172.21.0.1:37384","upstream_remote_address":"127.0.0.1:10001"},"executed_plugins":[{"name":"debugMode","per_phase_cost_seconds":{"DecodeHeaders":0.000004}},{"name":"limitReq","per_phase_cost_seconds":{"DecodeHeaders":0.042762708}}]}
```

其中包含如下信息：

```json
{
    // 请求总耗时
    "total_seconds": 4.364525,
    "request": {
        "headers": {
            // 请求头
            ":authority": [
                "localhost:10000"
            ],
            ":method": [
                "HEAD"
            ],
            ":path": [
                "/echo"
            ],
            ":scheme": [
                "http"
            ],
            "user-agent": [
                "Go-http-client/1.1"
            ],
            "x-forwarded-proto": [
                "http"
            ],
            "x-request-id": [
                "13cd56d5-8ea8-4c9b-ad70-459b48b3195a"
            ]
        }
    },
    "response": {
        "headers": {
            // 响应头（如果有）
            ":status": [
                "200"
            ],
            "date": [
                "Thu, 13 Jun 2024 09:58:37 GMT"
            ],
            "server": [
                "envoy"
            ],
            "transfer-encoding": [
                "chunked"
            ],
            "x-envoy-upstream-service-time": [
                "0"
            ]
        }
    },
    "stream_info": {
        // 客户端地址
        "downstream_remote_address": "172.21.0.1:48662",
        // 上游地址（如果有）
        "upstream_remote_address": "127.0.0.1:10001"
    },
    "executed_plugins": [
        // 执行的插件列表（如果有），以具体执行的顺序排序。
        // 注意因为 OnLog 阶段的时间不会算入请求耗时内，所以这里没有统计 OnLog 阶段执行的插件。
        // 另外如果客户端提前结束请求，可能会导致某些插件没有执行，或者执行完后没有上报统计数据。
        {
            "name": "debugMode",
            // 每个阶段耗时
            "per_phase_cost_seconds": {
                "DecodeHeaders": 0.000001876
            }
        },
        {
            "name": "limitReq",
            "per_phase_cost_seconds": {
                "DecodeHeaders": 0.041506417
            }
        }
    ]
}
```
