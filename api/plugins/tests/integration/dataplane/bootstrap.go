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

package dataplane

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

var (
	//go:embed bootstrap.yml
	boostrapTemplate []byte
)

type bootstrap struct {
	backendRoutes    []map[string]interface{}
	consumers        map[string]map[string]interface{}
	httpFilterGolang map[string]interface{}
	accessLogFormat  string
	clusters         []map[string]interface{}

	additionalFilterGolang map[string]map[string]interface{}

	dp *DataPlane
}

func Bootstrap() *bootstrap {
	return &bootstrap{
		backendRoutes: []map[string]interface{}{},
		consumers:     map[string]map[string]interface{}{},
		clusters:      []map[string]interface{}{},

		additionalFilterGolang: map[string]map[string]interface{}{},
	}
}

func (b *bootstrap) SetDataPlane(dp *DataPlane) {
	b.dp = dp
}

func (b *bootstrap) AddBackendRoute(s string) *bootstrap {
	var n map[string]interface{}
	err := yaml.Unmarshal([]byte(s), &n)
	if err != nil {
		panic(err)
	}
	b.backendRoutes = append(b.backendRoutes, n)
	return b
}

func (b *bootstrap) AddConsumer(name string, c map[string]interface{}) *bootstrap {
	if c["filters"] != nil {
		filters := c["filters"].(map[string]interface{})
		for _, f := range filters {
			d := map[string]interface{}{}
			fc := f.(map[string]interface{})["config"].(string)
			json.Unmarshal([]byte(fc), &d)
			f.(map[string]interface{})["config"] = d
		}
	}

	by, _ := json.Marshal(c)
	b.consumers[name] = map[string]interface{}{
		"v": rand.Intn(99999),
		"d": string(by),
	}
	return b
}

func (b *bootstrap) AddAdditionalFilterGolang(name string, c map[string]interface{}) *bootstrap {
	b.additionalFilterGolang[name] = c
	return b
}

func (b *bootstrap) SetFilterGolang(cfg map[string]interface{}) *bootstrap {
	b.httpFilterGolang = cfg
	return b
}

func (b *bootstrap) SetAccessLogFormat(fmt string) *bootstrap {
	b.accessLogFormat = fmt
	return b
}

func (b *bootstrap) AddCluster(s string) *bootstrap {
	var n map[string]interface{}
	err := yaml.Unmarshal([]byte(s), &n)
	if err != nil {
		panic(err)
	}
	b.clusters = append(b.clusters, n)
	return b
}

func (b *bootstrap) buildConfiguration() (map[string]interface{}, error) {
	var root map[string]interface{}
	// check if the input is valid yaml
	err := yaml.Unmarshal(boostrapTemplate, &root)
	if err != nil {
		return nil, err
	}

	// TODO: simplify it with some third party lib if possible
	backendHCM := root["static_resources"].(map[string]interface{})["listeners"].([]interface{})[1].(map[string]interface{})["filter_chains"].([]interface{})[0].(map[string]interface{})["filters"].([]interface{})[0].(map[string]interface{})["typed_config"].(map[string]interface{})
	vh := backendHCM["route_config"].(map[string]interface{})["virtual_hosts"].([]interface{})[0].(map[string]interface{})
	routes := vh["routes"].([]interface{})
	for _, backendRoute := range b.backendRoutes {
		routes = append(routes, backendRoute)
	}
	vh["routes"] = routes

	hcm := root["static_resources"].(map[string]interface{})["listeners"].([]interface{})[0].(map[string]interface{})["filter_chains"].([]interface{})[0].(map[string]interface{})["filters"].([]interface{})[0].(map[string]interface{})["typed_config"].(map[string]interface{})
	httpFilters := hcm["http_filters"].([]interface{})

	var cf map[string]interface{}
	for _, hf := range httpFilters {
		if hf.(map[string]interface{})["name"] == "htnn-consumer" {
			cf = hf.(map[string]interface{})["typed_config"].(map[string]interface{})
		}
	}

	consumers := cf["plugin_config"].(map[string]interface{})["value"].(map[string]interface{})["ns"].(map[string]interface{})
	for name, c := range b.consumers {
		consumers[name] = c
	}

	if b.httpFilterGolang != nil {
		for _, hf := range httpFilters {
			if hf.(map[string]interface{})["name"] == "htnn.filters.http.golang" {
				wrapper := map[string]interface{}{
					"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
					"value": b.httpFilterGolang,
				}
				hf.(map[string]interface{})["disabled"] = false
				hf.(map[string]interface{})["typed_config"].(map[string]interface{})["plugin_config"] = wrapper
			}
		}
	}
	if b.additionalFilterGolang != nil {
		fmt.Println("XXXXXXXXXXXX add additional filter")
		var additionalFilters []interface{}
		for name, cfg := range b.additionalFilterGolang {
			var found = false
			wrapper := map[string]interface{}{
				"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
				"value": cfg,
			}
			for _, hf := range httpFilters {
				if hf.(map[string]interface{})["name"] == name {
					hf.(map[string]interface{})["disabled"] = false
					hf.(map[string]interface{})["typed_config"].(map[string]interface{})["plugin_config"] = wrapper
					found = true
				}
			}
			if !found {
				additionalFilters = append(additionalFilters, map[string]interface{}{
					"name":     name,
					"disabled": false,
					"typed_config": map[string]interface{}{
						"@type":         "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
						"library_path":  "/etc/libgolang.so",
						"library_id":    "fm",
						"plugin_name":   name,
						"plugin_config": wrapper,
					},
				})
			}
		}
		httpFilters = append(additionalFilters, httpFilters...)
		hcm["http_filters"] = httpFilters
	}

	if b.accessLogFormat != "" {
		accessLog := hcm["access_log"].([]interface{})[0].(map[string]interface{})["typed_config"].(map[string]interface{})
		accessLog["log_format"] = map[string]interface{}{
			"text_format_source": map[string]interface{}{
				"inline_string": b.accessLogFormat + "\n",
			},
		}
	}

	staticResources := root["static_resources"].(map[string]interface{})
	clusters := staticResources["clusters"].([]interface{})

	port := 9999
	portEnv, _ := strconv.Atoi(os.Getenv("TEST_ENVOY_CONTROL_PLANE_PORT"))
	if portEnv != 0 {
		port = portEnv
	}
	for _, c := range clusters {
		if c.(map[string]interface{})["name"] == "config_server" {
			load := c.(map[string]interface{})["load_assignment"].(map[string]interface{})["endpoints"].([]interface{})[0].(map[string]interface{})["lb_endpoints"].([]interface{})[0].(map[string]interface{})["endpoint"].(map[string]interface{})["address"].(map[string]interface{})["socket_address"].(map[string]interface{})
			load["port_value"] = port
			break
		}
	}

	newClusters := []interface{}{}
	for _, c := range b.clusters {
		newClusters = append(newClusters, c)
	}
	staticResources["clusters"] = append(clusters, newClusters...)

	addr := root["admin"].(map[string]interface{})["address"].(map[string]interface{})["socket_address"].(map[string]interface{})
	addr["port_value"] = b.dp.AdminAPIPort()
	addr = staticResources["listeners"].([]interface{})[0].(map[string]interface{})["address"].(map[string]interface{})["socket_address"].(map[string]interface{})
	addr["port_value"] = b.dp.Port()

	if isBinaryMode() {
		for _, l := range root["static_resources"].(map[string]interface{})["listeners"].([]interface{}) {
			listener := l.(map[string]interface{})
			hcm := listener["filter_chains"].([]interface{})[0].(map[string]interface{})["filters"].([]interface{})[0].(map[string]interface{})["typed_config"].(map[string]interface{})
			httpFilters := hcm["http_filters"].([]interface{})
			for _, hf := range httpFilters {
				cfg := hf.(map[string]interface{})["typed_config"].(map[string]interface{})
				if cfg["@type"] == "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config" {
					cfg["library_path"] = b.dp.soPath
				}
			}
		}

		for _, c := range staticResources["clusters"].([]interface{}) {
			load := c.(map[string]interface{})["load_assignment"].(map[string]interface{})["endpoints"].([]interface{})[0].(map[string]interface{})["lb_endpoints"].([]interface{})[0].(map[string]interface{})["endpoint"].(map[string]interface{})["address"].(map[string]interface{})["socket_address"].(map[string]interface{})
			if load["address"] == "host.docker.internal" {
				load["address"] = "0.0.0.0"
			}
		}
	}

	return root, nil
}

func (b *bootstrap) WriteTo(cfgFile *os.File) error {
	root, err := b.buildConfiguration()
	if err != nil {
		return err
	}

	res, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}

	_, err = cfgFile.Write(res)
	return err
}

func (b *bootstrap) WriteToForValidation(cfgFile *os.File) error {
	root, err := b.buildConfiguration()
	if err != nil {
		return err
	}

	for _, l := range root["static_resources"].(map[string]interface{})["listeners"].([]interface{}) {
		listener := l.(map[string]interface{})
		if listener["name"] == "dynamic_config" {
			listener["internal_listener"] = nil
			listener["address"] = map[string]interface{}{
				"pipe": map[string]interface{}{
					"path": "/tmp/fake_socket_to_pass_validation",
				},
			}
		}
	}

	res, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}

	_, err = cfgFile.Write(res)
	return err
}
