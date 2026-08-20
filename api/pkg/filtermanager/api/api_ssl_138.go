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

//go:build envoy1.38 || envoy1.39 || envoydev

package api

import "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

// SslConnection provides SSL/TLS connection information for the downstream connection.
// On Envoy 1.38+, this is a type alias to the envoy SDK's SslConnection.
type SslConnection = api.SslConnection
