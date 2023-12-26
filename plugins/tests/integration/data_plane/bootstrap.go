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
	_ "embed"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	//go:embed bootstrap.yml
	boostrapTemplate []byte
)

type bootstrap struct {
	backendRoutes []map[string]interface{}
}

func Bootstrap() *bootstrap {
	return &bootstrap{
		backendRoutes: []map[string]interface{}{},
	}
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

func (b *bootstrap) WriteTo(cfgFile *os.File) error {
	var root map[string]interface{}
	// check if the input is valid yaml
	err := yaml.Unmarshal(boostrapTemplate, &root)
	if err != nil {
		return err
	}

	// TODO: simplify it with some third party lib if possible
	vh := root["static_resources"].(map[string]interface{})["listeners"].([]interface{})[1].(map[string]interface{})["filter_chains"].([]interface{})[0].(map[string]interface{})["filters"].([]interface{})[0].(map[string]interface{})["typed_config"].(map[string]interface{})["route_config"].(map[string]interface{})["virtual_hosts"].([]interface{})[0].(map[string]interface{})
	routes := vh["routes"].([]interface{})
	for _, backendRoute := range b.backendRoutes {
		routes = append(routes, backendRoute)
	}
	vh["routes"] = routes

	res, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}
	_, err = cfgFile.Write(res)
	return err
}
