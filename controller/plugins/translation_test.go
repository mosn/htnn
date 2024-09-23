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

package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/yaml"

	"mosn.io/htnn/controller/internal/istio"
	"mosn.io/htnn/controller/internal/translation"
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
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

// snakeToCamel converts a snake_case string to a camelCase string.
func snakeToCamel(s string) string {
	words := strings.Split(s, "_")
	for i := 1; i < len(words); i++ {
		words[i] = cases.Title(language.Und, cases.NoLower).String(words[i])
	}
	return strings.Join(words, "")
}

func TestHTTPPlugins(t *testing.T) {
	inputFiles, err := filepath.Glob(filepath.Join("testdata", "http", "*.in.yml"))
	require.NoError(t, err)

	var vs *istiov1a3.VirtualService
	var gw *istiov1a3.Gateway
	input := []map[string]interface{}{}
	mustUnmarshal(t, filepath.Join("testdata", "http", "default.yml"), &input)

	for _, in := range input {
		obj := pkg.MapToObj(in)
		gvk := obj.GetObjectKind().GroupVersionKind()
		if gvk.Kind == "VirtualService" {
			vs = obj.(*istiov1a3.VirtualService)
		} else if gvk.Group == "networking.istio.io" && gvk.Kind == "Gateway" {
			gw = obj.(*istiov1a3.Gateway)
		}
	}

	for _, inputFile := range inputFiles {
		name := testName(inputFile)
		t.Run(name, func(t *testing.T) {
			var fp mosniov1.FilterPolicy
			mustUnmarshal(t, inputFile, &fp)

			s := translation.NewInitState()
			s.AddPolicyForVirtualService(&fp, vs, []*istiov1a3.Gateway{gw})

			fs, err := s.Process(context.Background())
			require.NoError(t, err)

			defaultEnvoyFilters := istio.DefaultEnvoyFilters()
			expPlugin := fmt.Sprintf("htnn.filters.http.%s", snakeToCamel(name))
			for key := range defaultEnvoyFilters {
				for _, ef := range fs.EnvoyFilters {
					if ef.Name == key.Name {
						if ef.Name == "htnn-http-filter" {
							kept := []*istioapi.EnvoyFilter_EnvoyConfigObjectPatch{}
							for _, cp := range ef.Spec.ConfigPatches {
								st := cp.Patch.Value
								name := st.AsMap()["name"].(string)
								if name == expPlugin {
									kept = append(kept, cp)
								}
							}
							ef.Spec.ConfigPatches = kept
						} else {
							delete(fs.EnvoyFilters, key)
						}
						break
					}
				}
			}

			var out []*istiov1a3.EnvoyFilter
			for _, ef := range fs.EnvoyFilters {
				// drop irrelevant fields
				ef.Labels = nil
				ef.Annotations = nil
				out = append(out, ef)
			}
			sort.Slice(out, func(i, j int) bool {
				if out[i].Namespace != out[j].Namespace {
					return out[i].Namespace < out[j].Namespace
				}
				return out[i].Name < out[j].Name
			})
			d, _ := yaml.Marshal(out)
			actual := string(d)

			outputFilePath := strings.ReplaceAll(inputFile, ".in.yml", ".out.yml")
			d, _ = os.ReadFile(outputFilePath)
			want := string(d)
			require.Equal(t, want, actual)
		})
	}
}

func TestNetworkPlugins(t *testing.T) {
	inputFiles, err := filepath.Glob(filepath.Join("testdata", "network", "*.in.yml"))
	require.NoError(t, err)

	var gw *istiov1a3.Gateway
	input := []map[string]interface{}{}
	mustUnmarshal(t, filepath.Join("testdata", "network", "default.yml"), &input)

	for _, in := range input {
		obj := pkg.MapToObj(in)
		gvk := obj.GetObjectKind().GroupVersionKind()
		if gvk.Group == "networking.istio.io" && gvk.Kind == "Gateway" {
			gw = obj.(*istiov1a3.Gateway)
		}
	}

	for _, inputFile := range inputFiles {
		name := testName(inputFile)
		t.Run(name, func(t *testing.T) {
			var hfp mosniov1.FilterPolicy
			mustUnmarshal(t, inputFile, &hfp)

			s := translation.NewInitState()
			s.AddPolicyForIstioGateway(&hfp, gw)

			fs, err := s.Process(context.Background())
			require.NoError(t, err)

			defaultEnvoyFilters := istio.DefaultEnvoyFilters()
			for key := range defaultEnvoyFilters {
				for _, ef := range fs.EnvoyFilters {
					if ef.Name == key.Name {
						delete(fs.EnvoyFilters, key)
					}
				}
			}
			for _, ef := range fs.EnvoyFilters {
				// drop irrelevant fields
				ef.Labels = nil
				ef.Annotations = nil
			}

			var out []*istiov1a3.EnvoyFilter
			for _, ef := range fs.EnvoyFilters {
				out = append(out, ef)
			}
			sort.Slice(out, func(i, j int) bool {
				if out[i].Namespace != out[j].Namespace {
					return out[i].Namespace < out[j].Namespace
				}
				return out[i].Name < out[j].Name
			})
			d, _ := yaml.Marshal(out)
			actual := string(d)

			outputFilePath := strings.ReplaceAll(inputFile, ".in.yml", ".out.yml")
			d, _ = os.ReadFile(outputFilePath)
			want := string(d)
			require.Equal(t, want, actual)
		})
	}
}
