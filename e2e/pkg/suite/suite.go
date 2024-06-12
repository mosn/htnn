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
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/gateway-api/conformance/utils/roundtripper"

	"mosn.io/htnn/e2e/pkg/k8s"
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

	forwarders []*exec.Cmd
	t          *testing.T
}

type Options struct {
	Client    client.Client
	Clientset *kubernetes.Clientset
}

func New(opt Options) *Suite {
	return &Suite{
		Opt:        opt,
		forwarders: []*exec.Cmd{},
	}
}

func (suite *Suite) Run(t *testing.T) {
	k8s.Prepare(t, suite.Opt.Client, "base/default.yml")
	k8s.Prepare(t, suite.Opt.Client, "base/nacos.yml")
	suite.waitNacos(t)
	suite.startPortForward(t)
	defer suite.stopPortForward(t)
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					fmt.Printf("panic in test %s: %v\n%s", test.Name, p, debug.Stack())
					t.Fail()
				}
			}()
			t.Logf("CleanUp test %q at %v", test.Name, time.Now())
			suite.cleanup(t)
			for _, manifest := range test.Manifests {
				k8s.Prepare(t, suite.Opt.Client, manifest)
			}
			// TODO: find a signal to indicate that it's OK to test.
			// EnvoyFilter is created doesn't mean that the RDS takes effects in Envoy.
			time.Sleep(500 * time.Millisecond)
			// TODO: configure Istio to push aggressively

			t.Logf("Run test %q at %v", test.Name, time.Now())
			suite.t = t // store t for logging
			test.Run(t, suite)
		})
	}
}

func (suite *Suite) waitNacos(t *testing.T) {
	cmdline := "kubectl wait --timeout=5m -n e2e deployment/nacos --for=condition=Available"
	cmd := strings.Fields(cmdline)
	wait := exec.Command(cmd[0], cmd[1:]...)
	err := wait.Run()
	require.NoError(t, err)
}

// We use port-forward so that both Linux and Mac can expose port in the same way
func (suite *Suite) startPortForward(t *testing.T) {
	// TODO: rewrite 'kubectl wait' with Go code
	for _, cond := range []struct {
		name string
		ns   string
	}{
		{name: "istio-ingressgateway", ns: k8s.IstioRootNamespace},
		{name: "default-istio", ns: k8s.DefaultNamespace},
		{name: "default-istio", ns: k8s.AnotherNamespace},
	} {
		cmdline := fmt.Sprintf("kubectl wait --timeout=5m -n %s deployment/%s --for=condition=Available",
			cond.ns, cond.name)
		cmd := strings.Fields(cmdline)
		wait := exec.Command(cmd[0], cmd[1:]...)
		err := wait.Run()
		require.NoError(t, err, "wait for deployment %s in namespace %s", cond.name, cond.ns)
	}

	cmdline := "./port-forward.sh"
	dests := []string{"istio-ingressgateway", "k8s-gateway-api", "k8s-gateway-api-another"}
	for _, d := range dests {
		forwarder := exec.Command(cmdline, d)
		forwarder.Stdout = os.Stdout
		forwarder.Stderr = os.Stderr
		err := forwarder.Start()
		require.NoError(t, err)
		suite.forwarders = append(suite.forwarders, forwarder)
	}
	time.Sleep(2 * time.Second) // wait for port-forward to take effect
	t.Log("port-forward started")
}

func (suite *Suite) stopPortForward(t *testing.T) {
	for _, fwd := range suite.forwarders {
		err := fwd.Process.Signal(os.Interrupt)
		require.NoError(t, err)
	}
	t.Log("port-forward stopped")
}

func (suite *Suite) cleanup(t *testing.T) {
	k8s.CleanUp(t, suite.Opt.Client)
}

func (suite *Suite) K8sClient() client.Client {
	return suite.Opt.Client
}

func (suite *Suite) Head(path string, header http.Header) (*http.Response, error) {
	return suite.do("HEAD", path, header, nil)
}

func (suite *Suite) Options(path string, header http.Header) (*http.Response, error) {
	return suite.do("OPTIONS", path, header, nil)
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
	suite.t.Logf("send HTTP request %s %s at %v", method, path, time.Now())
	req, err := http.NewRequest(method, "http://localhost:10000"+path, body)
	if err != nil {
		return nil, err
	}
	req.Header = header
	tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
		return net.DialTimeout("tcp", ":10000", 1*time.Second)
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

	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
