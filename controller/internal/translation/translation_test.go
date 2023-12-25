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

package translation

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/yaml"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/controller/internal/istio"
)

func testName(inputFile string) string {
	_, fileName := filepath.Split(inputFile)
	return strings.TrimSuffix(fileName, ".in.yml")
}

func mustUnmarshal(t *testing.T, fn string, out interface{}) {
	input, err := os.ReadFile(fn)
	require.NoError(t, err)
	require.NoError(t, yaml.UnmarshalStrict(input, out, yaml.DisallowUnknownFields))
}

type testInput struct {
	// we use sigs.k8s.io/yaml which uses JSON under the hover
	HTTPFilterPolicy map[string][]*mosniov1.HTTPFilterPolicy `json:"httpFilterPolicy"`

	VirtualService map[string][]*istiov1b1.VirtualService `json:"virtualService"`
	IstioGateway   []*istiov1b1.Gateway                   `json:"istioGateway"`

	HTTPRoute map[string][]*gwapiv1.HTTPRoute `json:"httpRoute"`
	Gateway   []*gwapiv1.Gateway              `json:"gateway"`
}

func TestTranslate(t *testing.T) {
	inputFiles, err := filepath.Glob(filepath.Join("testdata", "translation", "*.in.yml"))
	require.NoError(t, err)

	for _, inputFile := range inputFiles {
		inputFile := inputFile
		t.Run(testName(inputFile), func(t *testing.T) {
			input := &testInput{}
			mustUnmarshal(t, inputFile, input)

			s := NewInitState(nil)

			// set up resources
			for _, gw := range input.Gateway {
				hrs := input.HTTPRoute[gw.Name]
				for _, hr := range hrs {
					hfps := input.HTTPFilterPolicy[hr.Name]
					for _, hfp := range hfps {
						// fulfill default fields
						if hfp.Namespace == "" {
							hfp.SetNamespace("default")
						}
						if hr.Namespace == "" {
							hr.SetNamespace("default")
						}
						if gw.Namespace == "" {
							gw.SetNamespace("default")
						}
						s.AddPolicyForHTTPRoute(hfp, hr, gw)
					}
				}
			}
			for _, gw := range input.IstioGateway {
				vss := input.VirtualService[gw.Name]
				for _, vs := range vss {
					hfps := input.HTTPFilterPolicy[vs.Name]
					for _, hfp := range hfps {
						// fulfill default fields
						if hfp.Namespace == "" {
							hfp.SetNamespace("default")
						}
						if vs.Namespace == "" {
							vs.SetNamespace("default")
						}
						if gw.Namespace == "" {
							gw.SetNamespace("default")
						}
						s.AddPolicyForVirtualService(hfp, vs, gw)
					}
				}
			}

			fs, err := s.Process(context.Background())
			require.NoError(t, err)

			defaultEnvoyFilters := istio.DefaultEnvoyFilters()
			for name := range defaultEnvoyFilters {
				found := false
				for _, ef := range fs.EnvoyFilters {
					if ef.Name == name {
						found = true
						delete(fs.EnvoyFilters, name)
						break
					}
				}
				require.True(t, found)
			}

			var out []*istiov1a3.EnvoyFilter
			for _, ef := range fs.EnvoyFilters {
				out = append(out, ef)
			}
			sort.Slice(out, func(i, j int) bool {
				return out[i].Name < out[j].Name
			})
			d, _ := yaml.Marshal(out)
			actual := string(d)

			outputFilePath := strings.ReplaceAll(inputFile, ".in.yml", ".out.yml")
			d, _ = os.ReadFile(outputFilePath)
			want := string(d)
			// google/go-cmp is not used here as it will compare unexported fields by default.
			// Calling IgnoreUnexported for each types in istio object is too cubmersome so we
			// just use string comparison here.
			require.Equal(t, want, actual)
		})
	}
}
