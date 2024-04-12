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

package component

import (
	"context"

	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Output interface {
	FromHTTPFilterPolicy(ctx context.Context, envoyFilters map[string]*istiov1a3.EnvoyFilter) error
	FromConsumer(ctx context.Context, envoyFilter *istiov1a3.EnvoyFilter) error
	// FromServiceRegistry writes the generated ServiceEntries to the output. Unlike the other generators,
	// it assumes the write already succeed, and don't retry on error,
	// so the output should handle the retry by themselves. That's why the error is not returned here.
	FromServiceRegistry(ctx context.Context, serviceEntries map[string]*istioapi.ServiceEntry)
}

type ResourceManager interface {
	Get(ctx context.Context, key client.ObjectKey, out client.Object) error
	List(ctx context.Context, list client.ObjectList) error
	UpdateStatus(ctx context.Context, obj client.Object, statusPtr any) error
}

type CtrlLogger interface {
	Error(msg any)
	Errorf(format string, args ...any)
	Info(msg any)
	Infof(format string, args ...any)
}
