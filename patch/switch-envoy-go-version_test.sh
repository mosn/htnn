#!/usr/bin/env bash
# Copyright The HTNN Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

mkdir -p "$work_dir/api" "$work_dir/plugins" "$work_dir/patch"
cp "$repo_root/api/go.mod" "$work_dir/api/go.mod"
cp "$repo_root/plugins/go.mod" "$work_dir/plugins/go.mod"
cp "$repo_root/patch/switch-envoy-go-version.sh" "$work_dir/patch/"

pushd "$work_dir" >/dev/null
./patch/switch-envoy-go-version.sh 1.38.0
./patch/switch-envoy-go-version.sh 1.39.0
./patch/switch-envoy-go-version.sh 1.39.0

for go_mod in api/go.mod plugins/go.mod; do
    test "$(grep -c '^replace github.com/envoyproxy/envoy ' "$go_mod")" -eq 1
    grep -qx 'replace github.com/envoyproxy/envoy => github.com/envoyproxy/envoy v1.39.0' "$go_mod"
    grep -qx 'require github.com/envoyproxy/go-control-plane/contrib v1.36.0' "$go_mod"
done

./patch/switch-envoy-go-version.sh 1.32.0
for go_mod in api/go.mod plugins/go.mod; do
    test "$(grep -c '^replace github.com/envoyproxy/envoy ' "$go_mod" || true)" -eq 0
    test "$(grep -c '^require github.com/envoyproxy/go-control-plane/contrib ' "$go_mod" || true)" -eq 0
done
popd >/dev/null
