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

package procession

import (
	"context"

	istioapi "istio.io/api/networking/v1beta1"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

type Output interface {
	FromHTTPFilterPolicy(ctx context.Context, envoyFilters map[string]*istiov1a3.EnvoyFilter) error
	FromConsumer(ctx context.Context, envoyFilter *istiov1a3.EnvoyFilter) error
	// FromServiceRegistry writes the generated ServiceEntries to the output. Unlike the other generators,
	// it assumes the write already succeed, and don't retry on error,
	// so the output should handle the retry by themselves. That's why the error is not returned here.
	FromServiceRegistry(ctx context.Context, serviceEntries map[string]*istioapi.ServiceEntry)
}
