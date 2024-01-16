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

package k8s

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("k8s")
)

const (
	DefaultNamespace = "e2e"
)

func Prepare(t *testing.T, client client.Client, source string) {
	data, err := os.ReadFile(source)
	require.NoError(t, err)
	input := bytes.NewBuffer(data)
	decoder := yaml.NewYAMLOrJSONDecoder(input, 4096)

	resources, err := readResources(decoder)
	if err != nil {
		require.NoErrorf(t, err, "error parsing manifest", "manifest: %s", input.String())
	}

	for _, res := range resources {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		namespacedName := types.NamespacedName{Namespace: res.GetNamespace(), Name: res.GetName()}
		fetchedObj := res.DeepCopy()
		err := client.Get(ctx, namespacedName, fetchedObj)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				require.NoErrorf(t, err, "error getting resource")
			}
			logger.Info("Creating", "name", res.GetName(), "kind", res.GetKind())
			err = client.Create(ctx, res.DeepCopy())
			require.NoErrorf(t, err, "error creating resource")
			continue
		}

		res.SetResourceVersion(fetchedObj.GetResourceVersion())
		logger.Info("Updating", "name", res.GetName(), "kind", res.GetKind())
		err = client.Update(ctx, res.DeepCopy())
		require.NoErrorf(t, err, "error updating resource")
	}
}

func readResources(decoder *yaml.YAMLOrJSONDecoder) ([]unstructured.Unstructured, error) {
	var resources []unstructured.Unstructured

	for {
		uObj := unstructured.Unstructured{}
		if err := decoder.Decode(&uObj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if len(uObj.Object) == 0 {
			continue
		}

		ns := uObj.GetNamespace()
		if ns == "" {
			uObj.SetNamespace(DefaultNamespace)
		}
		resources = append(resources, uObj)
	}

	return resources, nil
}
