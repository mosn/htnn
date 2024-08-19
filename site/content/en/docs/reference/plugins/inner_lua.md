---
title: Inner Lua
---

## Description

The `innerLua` plugin is the same as the `outerLua` plugin, but runs around proxying the request to the upstream cluster.

Because Envoy uses the onion model to proxy requests, the execution order is:

1. request starts
2. running other plugins
3. running `innerLua` and other plugins in `Inner` group
4. proxy to the upstream
5. running `innerLua` and other plugins in `Inner` group with the response
6. running other plugins with the response
7. request ends

Please refer to [outerLua](./outer_lua.md) plugin documentation to know how to use it. Don't forget to replace `outerLua` with `innerLua` when testing the example.

## Attribute

|       |         |
|-------|---------|
| Type  | General |
| Order | Inner   |
