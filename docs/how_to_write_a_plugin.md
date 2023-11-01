# How to write a plugin

1. Create a directory under `./plugins/`.
2. Think about the configuration and write down it into `./plugins/$your_plugin/config.pb`. Then run `make gen-proto`.
3. Finish the plugin. Don't forget to write tests. You can take `./plugins/demo` as the example.
4. Add the doc of the plugin in the `./plugins/$your_plugin/README.md`.
5. Run `make build-so`. Now the plugin is compiled into `libgolang.so`.
6. Add integration test in the `./tests/integration/plugins`. For how to run the integration test, please read `./tests/integration/plugins/README.md`.
