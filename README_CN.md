# HTNN

**构建状态**

[![test](https://github.com/mosn/htnn/actions/workflows/test.yml/badge.svg)](https://github.com/mosn/htnn/actions/workflows/test.yml)

**代码质量**

[![coverage](https://codecov.io/gh/mosn/htnn/branch/main/graph/badge.svg)](https://codecov.io/gh/mosn/htnn)
[![go report card](https://goreportcard.com/badge/github.com/mosn/htnn)](https://goreportcard.com/report/github.com/mosn/htnn)

---

HTNN（Hyper Trust-Native Network）是蚂蚁集团内部自研的云原生跨层网络解决方案，基于 Envoy 和 Istio 构建，支持通过 Go 运行时进行扩展。HTNN 在架构上拥抱云原生标准，支持多集群管理和灵活的可扩展性，形成了一套高研发效率的生态。通过开源 HTNN，蚂蚁集团希望社区能共享产品能力，共同打造先进的网络产品。

## 文档

* [介绍](https://github.com/mosn/htnn/blob/main/site/content/zh-hans/docs/getting-started/introduction.md)
* [快速开始](https://github.com/mosn/htnn/blob/main/site/content/zh-hans/docs/getting-started/quick_start.md)
* [参与贡献](https://github.com/mosn/htnn/blob/main/site/content/zh-hans/docs/developer-guide/get_involved.md)

如果你只需要使用 HTNN 的数据面来扩展 Envoy，请阅读：

* [数据面多版本支持](https://github.com/mosn/htnn/blob/main/site/content/zh-hans/docs/developer-guide/dataplane_support.md)

## 多版本 Envoy 与 Go 支持

HTNN 通过 build tag 机制支持多个 Envoy 版本。数据面的 Go 代码可编译为 shared library，加载到不同版本的 Envoy 中运行。

### 支持版本一览

| Envoy 版本 | Build Tag | 最低 Go 版本 | Envoy SDK（go.mod 中 replace） |
|------------|-----------|-------------|-------------------------------|
| dev（固定快照） | `envoydev` | 1.24.6 | `v1.38.0` |
| 1.39（仅外部数据面） | `envoy1.39` | 1.25 | `v1.39.0` |
| 1.38 | `envoy1.38` | 1.24.6 | `v1.38.0` |
| 1.37 | `envoy1.37` | 1.24.6 | `v1.37.2` |
| 1.36 | `envoy1.36` | 1.22 | `v1.36.6` |
| 1.35 | `envoy1.35` | 1.22 | `v1.35.3` |
| 1.32（默认） | _（无需 tag）_ | 1.22 | `v1.32.0`（go.mod 中已包含） |
| 1.31 | `envoy1.31` | 1.22 | `v1.31.x` |
| 1.29 | `envoy1.29` | 1.22 | `v1.29.x` |

### 外部用户（以模块方式引入 HTNN）

如果你在自己的项目中 import `mosn.io/htnn/api` 或 `mosn.io/htnn/plugins`：

#### 第一步：在你自己项目的 `go.mod` 中添加 replace 指令

```go.mod
module your-project

go 1.22

require (
    mosn.io/htnn/api v0.5.0
    mosn.io/htnn/plugins v0.5.0
)

// 指定目标 Envoy 版本为 1.39，添加以下行：
replace github.com/envoyproxy/envoy => github.com/envoyproxy/envoy v1.39.0
```

#### 第二步：使用对应的 build tag 编译

```bash
# Envoy 1.39（需要 Go 1.25 和 glibc 兼容的构建环境）：
CGO_ENABLED=1 go build -tags so,envoy1.39 --buildmode=c-shared -o libgolang.so .

# 默认版本（1.32），不需要额外 tag：
CGO_ENABLED=1 go build -tags so --buildmode=c-shared -o libgolang.so .

# 旧版本：
CGO_ENABLED=1 go build -tags so,envoy1.29 --buildmode=c-shared -o libgolang.so .
```

### HTNN 开发者（在本仓库中开发）

如果你是 HTNN 本身的开发者，可以使用便捷脚本：

```bash
# 切换 SDK 版本（更新 api/go.mod 和 plugins/go.mod 中的 Envoy replace）
./patch/switch-envoy-go-version.sh 1.39.0

# 构建 shared library
cd plugins && ENVOY_API_VERSION=1.39 make build-so-local

# 运行测试
cd api && ENVOY_API_VERSION=1.39 make unit-test
```

> **注意**：`switch-envoy-go-version.sh` 仅用于 HTNN 仓库内部，会修改 `api/go.mod` 和 `plugins/go.mod`，请在干净工作区中使用。外部用户应直接编辑自己项目的 `go.mod`。

### Go 版本要求

- **Envoy 1.29 ~ 1.36**：Go 1.22+
- **Envoy 1.37 ~ 1.38**：Go 1.24.6+（由 Envoy 的 `go.mod` 强制要求）
- **Envoy 1.39**：Go 1.25+（由 Envoy 的 `go.mod` 强制要求）

### CI 测试矩阵

CI 在多个 Go 和 Envoy 版本组合上运行测试：

- **Go 版本**：1.22、1.23、1.24、1.25
- **Envoy 版本**：1.29、1.31、1.32、1.35、1.36、1.37、1.38、1.39、dev
- 不兼容的组合（如 Go 1.22 + Envoy 1.37）会自动排除。

### 如何支持未来的 Envoy 新版本

当有新的 Envoy 版本发布时：

1. **检查 API 兼容性**：对比 `contrib/golang/common/go/api/filter.go` 在新版本和当前最新支持版本之间的差异。如果 Go SDK API 无变化，可以复用现有 build tag 代码路径。

2. **添加 build tag**（如果 API 无变化，添加到现有 tag 列表中）：
    - `api/pkg/filtermanager/api/api_latest.go`：添加新 tag
    - `api/pkg/filtermanager/api_impl_latest.go`：添加新 tag
    - `api/pkg/filtermanager/api/api_131_132.go`：排除新 tag
    - `api/pkg/filtermanager/api_impl_131_132.go`：排除新 tag
    - 检查 `api_ssl_138.go`、`api_impl_138.go`、`api_impl_pre138.go` 等版本文件，确保新版本只会命中一个实现。

3. **更新脚本和 CI**：
    - `patch/switch-envoy-go-version.sh`：将发布版本加入支持列表
    - `.github/workflows/test.yml`：将版本加入 Envoy 矩阵，并配置所需 Go builder 和运行时镜像

4. **更新文档**：
   - `site/content/en/docs/developer-guide/dataplane_support.md`
   - `site/content/zh-hans/docs/developer-guide/dataplane_support.md`
   - `README.md` 和 `README_CN.md`

5. **如果存在 API breaking changes**：按照现有兼容层模式（参考 `api_impl_129.go`、`api_impl_131_132.go`）创建新的版本特定文件，使用专属 build tag 范围。

### 兼容性说明

- `go.mod` 中的**默认版本**始终为 **1.32**。用户如需其他版本，必须通过 `replace` + build tag 实现。
- Envoy 1.35 ~ 1.39 共享**相同的 Go SDK API**（经逐文件对比确认），使用同一套代码路径。
- Envoy 1.39 仅支持外部自管的 `envoyproxy/envoy:contrib-v1.39.0` 数据面及使用 Go 1.25、glibc 兼容环境构建的 shared library。默认 Istio 1.21.3 数据面镜像、Helm chart 和 controller 没有升级或验证为 Envoy 1.39。
- 排查外部 1.39 构建时，请一起检查 Go 版本、Envoy SDK replace、`envoy1.39` tag、目标架构、builder glibc 版本和 Envoy 启动日志；SDK 与运行时镜像必须使用相同 Envoy patch 版本。
- 在旧版 Envoy 上执行仅新版才有的接口时，兼容层会输出错误日志并返回空值，不会崩溃。

## 致谢

**HTNN** 的诞生离不开社区优秀开源项目的贡献，特别感谢：

* [Envoy](https://www.envoyproxy.io)
* [Istio](https://istio.io)
