This directory shows how to write HTNN Go plugin in your project.

To present how this work, follow the steps:

1. Run `make build-so && make run-plugin` to start Envoy.
2. `curl http://127.0.0.1:10000/`. It is expected to see `Your plugin is run`.

It's recommended to read through this project before writing your plugin.
