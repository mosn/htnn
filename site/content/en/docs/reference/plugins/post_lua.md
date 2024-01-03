---
title: Post Lua
---

## Description

This plugin is the same as the `pre_lua` plugin, but runs around proxying the request to the upstream cluster.

Because Envoy uses the onion model to proxy requests, the execution order is:

1. request starts
2. running other plugins
3. running post-lua and other plugins in `Post` group
4. proxy to the upstream
5. running post-lua and other plugins in `Post` group
6. running other plugins with the response
7. request ends

Please refer to [pre_lua](./pre_lua) plugin documentation to know how to use it. Don't forget to replace `pre_lua` with `post_lua` when testing the example.

## Attribute

|       |         |
|-------|---------|
| Type  | General |
| Order | Post    |