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
	"fmt"
	"maps"
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
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/yaml"

	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/istio"
	_ "mosn.io/htnn/controller/plugins"    // register plugins
	_ "mosn.io/htnn/controller/registries" // register registries
	"mosn.io/htnn/controller/tests/pkg"
	mosniov1 "mosn.io/htnn/types/apis/v1"
)

func init() {
	plugins.RegisterHttpPluginType("animal", &plugins.MockPlugin{})
	plugins.RegisterHttpPluginType("localReply", &plugins.MockPlugin{})
}

func testName(inputFile string) string {
	_, fileName := filepath.Split(inputFile)
	return strings.TrimSuffix(fileName, ".in.yml")
}

func mustUnmarshal(t *testing.T, fn string, out interface{}) {
	input, err := os.ReadFile(fn)
	require.NoError(t, err)
	require.NoError(t, yaml.UnmarshalStrict(input, out, yaml.DisallowUnknownFields))
}

type Features struct {
	EnableLDSPluginViaECDS   bool `json:"enableLDSPluginViaECDS"`
	UseWildcardIPv6InLDSName bool `json:"useWildcardIPv6InLDSName"`
}

type testInput struct {
	// we use sigs.k8s.io/yaml which uses JSON under the hover
	HTTPFilterPolicy map[string][]*mosniov1.HTTPFilterPolicy `json:"httpFilterPolicy"`

	VirtualService map[string][]*istiov1a3.VirtualService `json:"virtualService"`
	IstioGateway   []*istiov1a3.Gateway                   `json:"istioGateway"`

	HTTPRoute map[string][]*gwapiv1b1.HTTPRoute `json:"httpRoute"`
	Gateway   []*gwapiv1b1.Gateway              `json:"gateway"`

	Features *Features `json:"features"`
}

func TestTranslate(t *testing.T) {
	inputFiles, err := filepath.Glob(filepath.Join("testdata", "translation", "*.in.yml"))
	require.NoError(t, err)

	for _, inputFile := range inputFiles {
		t.Run(testName(inputFile), func(t *testing.T) {
			input := &testInput{}
			mustUnmarshal(t, inputFile, input)

			if input.Features != nil {
				feats := input.Features
				if feats.EnableLDSPluginViaECDS {
					os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "true")
				}
				if feats.UseWildcardIPv6InLDSName {
					os.Setenv("HTNN_USE_WILDCARD_IPV6_IN_LDS_NAME", "true")
				}
				config.Init()

				defer func() {
					if feats.EnableLDSPluginViaECDS {
						os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "false")
					}
					if feats.UseWildcardIPv6InLDSName {
						os.Setenv("HTNN_USE_WILDCARD_IPV6_IN_LDS_NAME", "false")
					}
					config.Init()
				}()
			}

			s := NewInitState()

			// set up resources
			type gwapiWrapper struct {
				hr  *gwapiv1b1.HTTPRoute
				gws []*gwapiv1b1.Gateway
			}
			hrToGws := map[string]gwapiWrapper{}
			for _, gw := range input.Gateway {
				// fulfill default fields
				if gw.Namespace == "" {
					gw.SetNamespace("default")
				}
				hrs := input.HTTPRoute[gw.Name]
				for _, hr := range hrs {
					if hr.Namespace == "" {
						hr.SetNamespace("default")
					}
					hrToGws[hr.Name] = gwapiWrapper{
						hr:  hr,
						gws: append(hrToGws[hr.Name].gws, gw),
					}
				}
			}
			for name, wrapper := range hrToGws {
				hfps := input.HTTPFilterPolicy[name]
				for _, hfp := range hfps {
					if hfp.Namespace == "" {
						hfp.SetNamespace("default")
					}
					s.AddPolicyForHTTPRoute(hfp, wrapper.hr, wrapper.gws)
				}
			}

			type istioWrapper struct {
				vs  *istiov1a3.VirtualService
				gws []*istiov1a3.Gateway
			}
			vsToGws := map[string]istioWrapper{}
			for _, gw := range input.IstioGateway {
				// fulfill default fields
				if gw.Namespace == "" {
					gw.SetNamespace("default")
				}
				vss := input.VirtualService[gw.Name]
				for _, vs := range vss {
					if vs.Namespace == "" {
						vs.SetNamespace("default")
					}
					vsToGws[vs.Name] = istioWrapper{
						vs:  vs,
						gws: append(vsToGws[vs.Name].gws, gw),
					}
				}
			}

			hfpsMap := maps.Clone(input.HTTPFilterPolicy)
			for name, wrapper := range vsToGws {
				hfps := hfpsMap[name]
				if hfps != nil {
					// Currently, a policy can only target one resource.
					delete(hfpsMap, name)
				}
				for _, hfp := range hfps {
					if hfp.Namespace == "" {
						hfp.SetNamespace("default")
					}
					s.AddPolicyForVirtualService(hfp, wrapper.vs, wrapper.gws)
				}
			}

			// For gateway-only cases
			for _, gw := range input.IstioGateway {
				name := gw.Name
				hfps := hfpsMap[name]
				for _, hfp := range hfps {
					if hfp.Namespace == "" {
						hfp.SetNamespace("default")
					}
					s.AddPolicyForIstioGateway(hfp, gw)
				}
			}
			if config.EnableLDSPluginViaECDS() {
				for _, gw := range input.IstioGateway {
					s.AddIstioGateway(gw)
				}
			}

			fs, err := s.Process(context.Background())
			require.NoError(t, err)

			defaultEnvoyFilters := istio.DefaultEnvoyFilters()
			for key := range defaultEnvoyFilters {
				found := false
				for _, ef := range fs.EnvoyFilters {
					if ef.Name == key.Name {
						found = true
						delete(fs.EnvoyFilters, key)
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
			// google/go-cmp is not used here as it will compare unexported fields by default.
			// Calling IgnoreUnexported for each types in istio object is too cubmersome so we
			// just use string comparison here.
			require.Equal(t, want, actual)
		})
	}
}

// snakeToCamel converts a snake_case string to a camelCase string.
func snakeToCamel(s string) string {
	words := strings.Split(s, "_")
	for i := 1; i < len(words); i++ {
		words[i] = cases.Title(language.Und, cases.NoLower).String(words[i])
	}
	return strings.Join(words, "")
}

func TestPlugins(t *testing.T) {
	inputFiles, err := filepath.Glob(filepath.Join("testdata", "plugins", "*.in.yml"))
	require.NoError(t, err)

	var vs *istiov1a3.VirtualService
	var gw *istiov1a3.Gateway
	input := []map[string]interface{}{}
	mustUnmarshal(t, filepath.Join("testdata", "plugins", "default.yml"), &input)

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
			var hfp mosniov1.HTTPFilterPolicy
			mustUnmarshal(t, inputFile, &hfp)

			s := NewInitState()
			s.AddPolicyForVirtualService(&hfp, vs, []*istiov1a3.Gateway{gw})

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
