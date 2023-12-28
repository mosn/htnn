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

package suite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/gateway-api/conformance/utils/roundtripper"

	mosniov1 "mosn.io/moe/controller/api/v1"
	"mosn.io/moe/e2e/pkg/k8s"
	"mosn.io/moe/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("suite")
)

type Test struct {
	Name      string
	Manifests []string
	Run       func(*testing.T, *Suite)
}

var (
	tests = []Test{}
)

func Register(test Test) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("unexpected error")
	}
	name := strings.TrimSuffix(filepath.Base(filename), ".go")
	test.Name = name
	test.Manifests = append(test.Manifests, filepath.Join("tests", name+".yml"))
	tests = append(tests, test)
}

type Suite struct {
	Opt Options

	forwarder *exec.Cmd
}

type Options struct {
	Client    client.Client
	Clientset *kubernetes.Clientset
}

func New(opt Options) *Suite {
	return &Suite{
		Opt: opt,
	}
}

func (suite *Suite) Run(t *testing.T) {
	k8s.Prepare(t, suite.Opt.Client, "base/default.yml")
	suite.startPortForward(t)
	defer suite.stopPortForward(t)
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			suite.cleanup(t)
			for _, manifest := range test.Manifests {
				k8s.Prepare(t, suite.Opt.Client, manifest)
			}
			// TODO: find a signal to indicate that it's OK to test.
			// EnvoyFilter is created doesn't mean that the RDS takes effects in Envoy.
			time.Sleep(500 * time.Millisecond)
			// TODO: configure Istio to push aggressively

			logger.Info("Run test", "name", test.Name)
			test.Run(t, suite)
		})
	}
}

// We use port-forward so that both Linux and Mac can expose port in the same way
func (suite *Suite) startPortForward(t *testing.T) {
	cmdline := "../ci/port-forward.sh"
	suite.forwarder = exec.Command(cmdline)
	suite.forwarder.Stdout = os.Stdout
	suite.forwarder.Stderr = os.Stderr
	err := suite.forwarder.Start()
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // wait for port-forward to take effect
	logger.Info("port-forward started")
}

func (suite *Suite) stopPortForward(t *testing.T) {
	if suite.forwarder != nil {
		err := suite.forwarder.Process.Signal(os.Interrupt)
		require.NoError(t, err)
		logger.Info("port-forward stopped")
	}
}

func (suite *Suite) cleanup(t *testing.T) {
	c := suite.Opt.Client
	ctx := context.Background()
	all := corev1.NamespaceAll
	listOps := &client.ListOptions{
		Namespace: all,
	}
	var policies mosniov1.HTTPFilterPolicyList
	if err := c.List(ctx, &policies, listOps); err == nil {
		for i := range policies.Items {
			e := policies.Items[i]
			require.NoError(t, c.Delete(ctx, &e))
			logger.Info("Deleted", "name", e.GetName(), "kind", e.GetObjectKind())
		}
	}

	var virtualservices istiov1b1.VirtualServiceList
	if err := c.List(ctx, &virtualservices); err == nil {
		for _, e := range virtualservices.Items {
			require.NoError(t, c.Delete(ctx, e))
			logger.Info("Deleted", "name", e.GetName(), "kind", e.GetObjectKind())
		}
	}
	// let HTNN to clean up EnvoyFilter
}

func (suite *Suite) Head(path string, header http.Header) (*http.Response, error) {
	return suite.do("HEAD", path, header, nil)
}

func (suite *Suite) Get(path string, header http.Header) (*http.Response, error) {
	return suite.do("GET", path, header, nil)
}

func (suite *Suite) Delete(path string, header http.Header) (*http.Response, error) {
	return suite.do("DELETE", path, header, nil)
}

func (suite *Suite) Post(path string, header http.Header, body io.Reader) (*http.Response, error) {
	return suite.do("POST", path, header, body)
}

func (suite *Suite) Put(path string, header http.Header, body io.Reader) (*http.Response, error) {
	return suite.do("PUT", path, header, body)
}

func (suite *Suite) Patch(path string, header http.Header, body io.Reader) (*http.Response, error) {
	return suite.do("PATCH", path, header, body)
}

func (suite *Suite) do(method string, path string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://default.local:18000"+path, body)
	if err != nil {
		return nil, err
	}
	req.Header = header
	tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
		return net.DialTimeout("tcp", ":18000", 1*time.Second)
	}}

	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	return resp, err
}

// Capture is modified from gateway-api's CapturedRequest, under Apache License 2.0.
func (suite *Suite) Capture(resp *http.Response) (*roundtripper.CapturedRequest, *roundtripper.CapturedResponse, error) {
	cReq := &roundtripper.CapturedRequest{}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.Header.Get("Content-type") == "application/json" {
		err = json.Unmarshal(body, cReq)
		if err != nil {
			return nil, nil, fmt.Errorf("unexpected error reading response: %w", err)
		}
	} else {
		return nil, nil, fmt.Errorf("unexpected content-type: %s", resp.Header.Get("Content-type"))
	}

	cRes := &roundtripper.CapturedResponse{
		StatusCode:    resp.StatusCode,
		ContentLength: resp.ContentLength,
		Protocol:      resp.Proto,
		Headers:       resp.Header,
	}

	return cReq, cRes, nil
}
