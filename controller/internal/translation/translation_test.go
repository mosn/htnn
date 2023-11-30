package translation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
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
	HTTPFilterPolicy []*mosniov1.HTTPFilterPolicy           `json:"httpFilterPolicy"`
	VirtualService   map[string][]*istiov1b1.VirtualService `json:"virtualService"`
	Gateway          map[string][]*istiov1b1.Gateway        `json:"gateway"`
}

type testOutput struct {
	// we use sigs.k8s.io/yaml which uses JSON under the hover
	ToUpdate []*istiov1a3.EnvoyFilter `json:"toUpdate,omitempty"`
	ToDelete []*istiov1a3.EnvoyFilter `json:"toDelete,omitempty"`
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
			for _, httpFilterPolicy := range input.HTTPFilterPolicy {
				vss := input.VirtualService[httpFilterPolicy.Name]
				for _, vs := range vss {
					gws := input.Gateway[vs.Name]
					for _, gw := range gws {
						// fulfill default fields
						if httpFilterPolicy.Namespace == "" {
							httpFilterPolicy.SetNamespace("default")
						}
						if vs.Namespace == "" {
							vs.SetNamespace("default")
						}
						if gw.Namespace == "" {
							gw.SetNamespace("default")
						}
						s.AddPolicyForVirtualService(httpFilterPolicy, vs, gw)
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
