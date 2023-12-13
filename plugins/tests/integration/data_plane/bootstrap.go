// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data_plane

import (
	"bytes"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

const (
	boostrapTemplate = `
node:
  id: id
  cluster: cluster

static_resources:
  listeners:
    - name: listener_0
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 10000
      per_connection_buffer_limit_bytes: 1024000
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                access_log:
                  - name: envoy.access_loggers.stdout
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
                http_filters:
                  - name: envoy.filters.http.golang
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config
                      library_id: fm
                      library_path: /etc/libgolang.so
                      plugin_name: fm
                  - name: envoy.filters.http.router
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
                rds:
                  route_config_name: dynamic_route
                  config_source:
                    resource_api_version: V3
                    api_config_source:
                      api_type: GRPC
                      transport_api_version: V3
                      grpc_services:
                        - envoy_grpc:
                            cluster_name: config_server
    - name: backend
      address:
        socket_address:
          address: 127.0.0.1
          port_value: 10001
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                access_log:
                  - name: envoy.access_loggers.stdout
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
                http_filters:
                  - name: envoy.filters.http.bandwidth_limit
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.bandwidth_limit.v3.BandwidthLimit
                      stat_prefix: bandwidth_limiter_custom_route
                      enable_mode: RESPONSE
                      limit_kbps: 1
                      # dummy config to satisfy Envoy validator, doesn't take effect
                    disabled: true
                  - name: envoy.filters.http.lua
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua
                  - name: envoy.filters.http.router
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: local_service
                      domains: ["*"]
                      routes:
                        - match:
                            path: /echo
                          direct_response:
                            status: 200
                            body:
                              inline_string: ""
                          typed_per_filter_config:
                            envoy.filters.http.lua:
                              "@type": type.googleapis.com/envoy.extensions.filters.http.lua.v3.LuaPerRoute
                              source_code:
                                inline_string: |
                                  function envoy_on_request(handle)
                                    local headers = handle:headers()
                                    local resp_headers = {[":status"] = "200"}
                                    for key, value in pairs(headers) do
                                      local k = "echo-" .. key
                                      local v = resp_headers[k]
                                      if v ~= nil then
                                        table.insert(v, value)
                                        value = v
                                      else
                                        value = {value}
                                      end
                                      resp_headers[k] = value
                                    end

                                    local always_wrap_body = true
                                    local body = handle:body(always_wrap_body)
                                    local size = body:length()
                                    local data = ""
                                    if size > 0 then
                                      data = body:getBytes(0, size)
                                    end

                                    handle:respond(
                                      resp_headers,
                                      data
                                    )
                                  end
                                  function envoy_on_response(handle)
                                  end
                        - match:
                            path_separated_prefix: /ext_auth
                          direct_response:
                            status: 200
                            body:
                              inline_string: ""
                          typed_per_filter_config:
                            envoy.filters.http.lua:
                              "@type": type.googleapis.com/envoy.extensions.filters.http.lua.v3.LuaPerRoute
                              source_code:
                                inline_string: |
                                  function envoy_on_request(handle)
                                    local always_wrap_body = true
                                    local body = handle:body(always_wrap_body)
                                    local size = body:length()
                                    local data = ""
                                    if size > 0 then
                                      data = body:getBytes(0, size)
                                    end

                                    local resp_headers = {[":status"] = "200"}
                                    resp_headers["body"] = data
                                    local headers = handle:headers()
                                    if headers:get("authorization") ~= "Basic amFjazIwMjE6MTIzNDU2" then
                                      resp_headers[":status"] = "403"
                                      resp_headers["reason"] = "not matched"
                                    end
                                    handle:respond(resp_headers)
                                  end
                                  function envoy_on_response(handle)
                                  end
                        - match:
                            path: /slow_resp
                          direct_response:
                            status: 200
                            body:
                              inline_string: ""
                          typed_per_filter_config:
                            envoy.filters.http.bandwidth_limit:
                              "@type": type.googleapis.com/envoy.extensions.filters.http.bandwidth_limit.v3.BandwidthLimit
                              stat_prefix: bandwidth_limiter_custom_route
                              enable_mode: RESPONSE
                              limit_kbps: 1
                            envoy.filters.http.lua:
                              "@type": type.googleapis.com/envoy.extensions.filters.http.lua.v3.LuaPerRoute
                              source_code:
                                inline_string: |
                                  function envoy_on_request(handle)
                                    local resp_headers = {[":status"] = "200"}
                                    local always_wrap_body = true
                                    local body = handle:body(always_wrap_body)
                                    local size = body:length()
                                    local data = ""
                                    if size > 0 then
                                      data = body:getBytes(0, size)
                                    end

                                    handle:respond(
                                      resp_headers,
                                      data
                                    )
                                  end
                                  function envoy_on_response(handle)
                                  end

  clusters:
    - name: backend
      type: strict_dns
      lb_policy: round_robin
      load_assignment:
        cluster_name: backend
        endpoints:
        - lb_endpoints:
          - endpoint:
              address:
                socket_address:
                  address: 127.0.0.1
                  port_value: 10001
    - name: config_server
      connect_timeout: 0.25s
      type: strict_dns
      lb_policy: round_robin
      http2_protocol_options: {}
      upstream_connection_options:
        tcp_keepalive: {}
      load_assignment:
        cluster_name: config_server
        endpoints:
        - lb_endpoints:
          - endpoint:
              address:
                socket_address:
                  address: host.docker.internal
                  port_value: 9999

admin:
  address:
    socket_address: { address: 0.0.0.0, port_value: 9998 }

`
)

type Bootstrap struct {
}

type cfgFileWrapper struct {
	*os.File
}

func (c *cfgFileWrapper) Write(p []byte) (n int, err error) {
	var obj interface{}
	// convert tab to space according to Go's default tabsize
	p = bytes.ReplaceAll(p, []byte("\t"), []byte("    "))
	// check if the input is valid yaml
	err = yaml.Unmarshal(p, &obj)
	if err != nil {
		return
	}
	res, err := yaml.Marshal(obj)
	if err != nil {
		return
	}
	return c.File.Write(res)
}

func WriteBoostrapConfig(cfgFile *os.File) error {
	tmpl, err := template.New("bootstrap").Parse(boostrapTemplate)
	if err != nil {
		return err
	}
	// TODO: design a group APIs and break down the config file.
	// So that people can register their paths or http filters, like test-nginx.
	return tmpl.Execute(&cfgFileWrapper{cfgFile}, &Bootstrap{})
}
