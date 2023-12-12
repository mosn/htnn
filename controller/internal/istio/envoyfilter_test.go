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

package istio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/yaml"
)

func TestDefaultFilters(t *testing.T) {
	out := []*istiov1a3.EnvoyFilter{}
	for _, ef := range DefaultEnvoyFilters() {
		out = append(out, ef)
	}
	d, _ := yaml.Marshal(out)
	actual := string(d)
	expFile := filepath.Join("testdata", "default_filters.yml")
	d, _ = os.ReadFile(expFile)
	want := string(d)
	require.Equal(t, want, actual)
}
