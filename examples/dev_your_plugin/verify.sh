#!/usr/bin/env bash
set -euo pipefail

make build-so
make run-plugin
sleep 5
curl http://0.0.0.0:10000 | grep "Your plugin is run"
make stop-plugin
