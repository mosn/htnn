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

package client

const (
	DefaultFetchPageSize = 1000
	DefaultNacosPort     = 8848
	DefaultTimeoutMs     = 5 * 1000
	DefaultLogLevel      = "warn"
	DefaultNotLoadCache  = true
	DefaultLogMaxDays    = 1
	DefaultLogMaxBackups = 10
	DefaultLogMaxSizeMB  = 1
)

type NacosService struct {
	GroupName   string
	ServiceName string
}

type Client interface {
	Subscribe(groupName string, serviceName string, callback func(services []SubscribeService, err error)) error
	Unsubscribe(groupName string, serviceName string, callback func(services []SubscribeService, err error)) error
	FetchAllServices() (map[NacosService]bool, error)
	GetNamespace() string
	GetGroups() []string
}

type SubscribeService struct {
	IP       string            `json:"ip"`
	Metadata map[string]string `json:"metadata"`
	Port     uint64            `json:"port"`
}
