package translation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
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
	EnvoyFilter      []*istiov1a3.EnvoyFilter               `json:"envoyFilter"`
}

type testOutput struct {
	// we use sigs.k8s.io/yaml which uses JSON under the hover
	ToUpdate []*istiov1a3.EnvoyFilter `json:"toUpdate,omitempty"`
	ToDelete []*istiov1a3.EnvoyFilter `json:"toDelete,omitempty"`
}

func TestTranslate(t *testing.T) {
	inputFiles, err := filepath.Glob(filepath.Join("testdata", "translation", "*.in.yml"))
	require.NoError(t, err)

	var toUpdate []*istiov1a3.EnvoyFilter
	var toDel []*istiov1a3.EnvoyFilter
	// We can't run the test in parallel as it contains a diff process. So it's safe to store delta to toUpdate
	patches := gomonkey.ApplyFunc(publishCustomResources,
		func(ctx *Ctx, addOrUpdate []*istiov1a3.EnvoyFilter, del []*istiov1a3.EnvoyFilter) error {
			toUpdate = addOrUpdate
			toDel = del
			return nil
		})
	defer patches.Reset()

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

			currentEnvoyFilters = map[string]*istiov1a3.EnvoyFilter{}
			for _, ef := range input.EnvoyFilter {
				currentEnvoyFilters[ef.Namespace+"/"+ef.Name] = ef
			}

			err := s.Process(context.Background())
			require.NoError(t, err)

			defaultEnvoyFilters := istio.DefaultEnvoyFilters()
			for name := range defaultEnvoyFilters {
				found := false
				for i, ef := range toUpdate {
					if ef.Name == name {
						found = true
						toUpdate = slices.Delete(toUpdate, i, i+1)
						break
					}
				}
				require.True(t, found)
			}

			out := &testOutput{
				ToUpdate: toUpdate,
				ToDelete: toDel,
			}
			d, _ := yaml.Marshal(out)
			actual := string(d)

			outputFilePath := strings.ReplaceAll(inputFile, ".in.yml", ".out.yml")
			d, _ = os.ReadFile(outputFilePath)
			want := string(d)
			// google/go-cmp is not used here as it will compare unexported fileds by default.
			// Calling IgnoreUnexported for each types in istio object is too cubmersome so we
			// just use string comparison here.
			require.Equal(t, want, actual)
		})
	}
}
