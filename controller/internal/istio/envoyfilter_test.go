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
