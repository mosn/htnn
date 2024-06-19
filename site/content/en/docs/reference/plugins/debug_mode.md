---
title: Debug Mode
---

## Description

The `debugMode` plugin is used to enable debug mode on the targeted Route.

## Attribute

|       |         |
|-------|---------|
| Type  | General |
| Order | Access  |

## Configuration

| Name    | Type    | Required | Validation | Description       |
|---------|---------|----------|------------|-------------------|
| slowLog | SlowLog | False    |            | Configuration for slow log |

### SlowLog

| Name      | Type                            | Required | Validation | Description                                                                 |
|-----------|---------------------------------|----------|------------|-----------------------------------------------------------------------------|
| threshold | [Duration](../../type#duration) | True     | > 0s       | If the request takes longer than this time, print an error log as shown below. |

## Usage

Assume we have the following HTTPRoute attached to `localhost:10000`, with a backend server listening on port `8080`:

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

### Slow log

Let's apply the following configuration:

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
    debugMode:
      config:
        slowLog:
          threshold: "1s"
```

When a request to route default takes longer than 1 second, the following error log will be printed:

```
[2024-06-14 03:31:38.868][30][error][golang] [contrib/golang/common/log/cgo.cc:24] slow log report: {"total_seconds":4.364525,"request":{"headers":{":autho
rity":["localhost:10000"],":method":["HEAD"],":path":["/echo"],":scheme":["http"],"user-agent":["Go-http-client/1.1"],"x-forwarded-proto":["http"],"x-request-id":["cb212874-58af-469c-b5a3-3bd0c70cb776"]}},"response":{"headers":{":status":["200"],"date":["Fri, 14 Jun 2024 03:31:38 GMT"],"server":["envoy"],"transfer-encoding":["chunked"],"x-envoy-upstream-service-time":["0"]}},"stream_info":{"downstream_remote_address":"172.21.0.1:37384","upstream_remote_address":"127.0.0.1:10001"},"executed_plugins":[{"name":"debugMode","per_phase_cost_seconds":{"DecodeHeaders":0.000004}},{"name":"limitReq","per_phase_cost_seconds":{"DecodeHeaders":0.042762708}}]}
```

Which contains information such as:

```json
{
    // request duration
    "total_seconds": 4.364525,
    "request": {
        "headers": {
            // request headers
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
    "response": { // response (if any)
        "headers": {
            // response headers
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
        // Client address
        "downstream_remote_address": "172.21.0.1:48662",
        // Upstream address (if any)
        "upstream_remote_address": "127.0.0.1:10001"
    },
    "executed_plugins": [
        // List of executed plugins (if any), ordered by their execution sequence.
        // Note that since the time spent in the OnLog phase is not counted into the request duration,
        // plugins executed during OnLog are not included here.
        {
            "name": "debugMode",
            // Per-phase duration
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
