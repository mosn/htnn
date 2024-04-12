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

package gatewayapi

import (
	"k8s.io/apimachinery/pkg/runtime"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type addToScheme func(s *runtime.Scheme) error

func AddToScheme(scheme *runtime.Scheme) error {
	fs := []addToScheme{
		gwapiv1b1.AddToScheme,
		gwapiv1.AddToScheme,
	}
	for _, f := range fs {
		if err := f(scheme); err != nil {
			return err
		}
	}
	return nil
}
