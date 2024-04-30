---
title: Inner Lua
---

## 说明

`innerLua` 插件与 `outerLua` 插件相同，但在向上游发起请求的前后运行。

因为 Envoy 使用洋葱模型来代理请求，执行顺序是：

1. 请求开始
2. 运行其他插件
3. 运行 `Inner` 组中的 `innerLua` 和其他插件
4. 代理到上游
5. 运行 `Inner` 组中的 `innerLua` 和其他插件处理响应
6. 运行其他插件处理响应
7. 请求结束

请参考 [outerLua](../outer_lua) 插件文档了解如何使用它。测试示例时不要忘记将 `outerLua` 替换为 `innerLua`。

## 属性

|       |         |
|-------|---------|
| Type  | General |
| Order | Inner   |
