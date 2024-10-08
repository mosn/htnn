---
title: Inner Ext Proc
---

## Description

The `innerExtProc` plugin is similar to the `outerExtProc` plugin but runs both before and after making upstream requests.

As Envoy uses an onion model to proxy requests, the execution order is:

1. Request start
2. Run other plugins
3. Run `innerExtProc` and other plugins in the `Inner` group
4. Proxy to upstream
5. Run `innerExtProc` and other plugins in the `Inner` group to process the response
6. Run other plugins to process the response
7. Request ends

Please refer to the [outerExtProc](./outer_ext_proc.md) plugin documentation to learn how to use it. Do not forget to replace `outerExtProc` with `innerExtProc` when testing examples.

## Attribute

|        |              |
|--------|--------------|
| Type   | General      |
| Order  | Inner        |
| Status | Experimental |
