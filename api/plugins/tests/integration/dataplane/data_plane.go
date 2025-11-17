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

package dataplane

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/log"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
)

var (
	logger = log.DefaultLogger.WithName("dataplane")

	testRootDirName = "test-envoy"
	containerName   = "run_envoy_for_test"

	validationCache = map[[16]byte]struct{}{}
)

type DataPlane struct {
	cmd  *exec.Cmd
	t    *testing.T
	opt  *Option
	done chan error

	latestRouteVersion string

	dataPlanePort string
	adminAPIPort  string
	soPath        string
}

type Option struct {
	LogLevel  string
	Envs      map[string]string
	Bootstrap *bootstrap

	NoErrorLogCheck    bool
	ExpectLogPattern   []string
	ExpectNoLogPattern []string
}

func addEnvironemntVariables(cmd *exec.Cmd, envs map[string]string) {
	for k, v := range envs {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
}

type serializableFileWriter struct {
	*os.File

	lock sync.Mutex
}

func (w *serializableFileWriter) Write(p []byte) (n int, err error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.File.Write(p)
}

func (w *serializableFileWriter) Unbox() *os.File {
	return w.File
}

func StartDataPlane(t *testing.T, opt *Option) (*DataPlane, error) {
	if opt == nil {
		opt = &Option{}
	}
	if opt.LogLevel == "" {
		opt.LogLevel = "info"
	}
	if opt.Bootstrap == nil {
		opt.Bootstrap = Bootstrap()
	}

	dp := &DataPlane{
		t:   t,
		opt: opt,
	}
	opt.Bootstrap.SetDataPlane(dp)

	err := dp.cleanup(t)
	if err != nil {
		return nil, err
	}

	dir := dp.root()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	coverDir := helper.CoverDir()
	_, err = os.Stat(coverDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if err := os.MkdirAll(coverDir, 0755); err != nil {
			return nil, err
		}
		// When the integration test is run with `go test ...`, the previous coverage files are not removed.
		// Since we only care about the coverage in CI, it is fine so far.
	}

	pwd, _ := os.Getwd()
	soPath := filepath.Join(pwd, "libgolang.so")
	st, err := os.Stat(soPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error(err, "Shared library not found. Please build the shared library before running integration test, for example calling `make build-test-so`",
				"shared library path", soPath)
		}
		return nil, err
	}
	if st.IsDir() {
		err := errors.New("bad shared library detected")
		logger.Error(err, "Please remove the bad file and build the shared library before running integration test, for example calling `make build-test-so`",
			"shared library path", soPath)
		return nil, err
	}
	dp.soPath = soPath

	adminAPIPort := "9998"
	adminAPIPortEnv := os.Getenv("TEST_ENVOY_ADMIN_API_PORT")
	if adminAPIPortEnv != "" {
		adminAPIPort = adminAPIPortEnv
	}
	dp.adminAPIPort = adminAPIPort

	dataPlanePort := "10000"
	dataPlanePortEnv := os.Getenv("TEST_ENVOY_DATA_PLANE_PORT")
	if dataPlanePortEnv != "" {
		dataPlanePort = dataPlanePortEnv
	}
	dp.dataPlanePort = dataPlanePort

	cfgFilename := "envoy.yaml"
	cfgFile, err := os.Create(filepath.Join(dir, cfgFilename))
	if err != nil {
		return nil, err
	}

	err = opt.Bootstrap.WriteTo(cfgFile)
	cfgFile.Close()
	if err != nil {
		return nil, err
	}

	var cmdline, envoyCmd string

	if isBinaryMode() {
		envoyCmd = binaryPath + " -c " + cfgFile.Name()

		if len(opt.Envs) == 0 {
			opt.Envs = map[string]string{}
		}
		opt.Envs["GOCOVERDIR"] = coverDir

	} else {
		hostAddr := ""
		if runtime.GOOS == "linux" {
			// We use this special domain to access the control plane on host.
			// It works with Docker for Win/Mac (--network host doesn't work).
			// For Linux's Docker, a special option is used instead
			hostAddr = "--add-host=host.docker.internal:host-gateway"
		}

		currentUser, err := user.Current()
		if err != nil {
			return nil, err
		}

		networkName := "services_service"
		err = exec.Command("docker", "network", "inspect", networkName).Run()
		if err != nil {
			logger.Info("docker network used by test not found, create one")
			err = exec.Command("docker", "network", "create", networkName).Run()
			if err != nil {
				return nil, err
			}
		}

		image := "m.daocloud.io/docker.io/envoyproxy/envoy:contrib-v1.32.0"

		specifiedImage := os.Getenv("PROXY_IMAGE")
		if specifiedImage != "" {
			image = specifiedImage
		}

		b, err := exec.Command("docker", "images", image).Output()
		if err != nil {
			return nil, err
		}
		if len(strings.Split(string(b), "\n")) < 3 {
			cmd := exec.Command("docker", "pull", image)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			logger.Info("pull envoy image", "cmdline", cmd.String())
			err = cmd.Run()
			if err != nil {
				return nil, err
			}
		}

		envs := []string{}
		for k, v := range opt.Envs {
			envs = append(envs, "-e", k+"="+v)
		}

		cmdline = "docker run" +
			" --name " + containerName +
			" --network " + networkName +
			" --user " + currentUser.Uid +
			" --rm -t -v " +
			cfgFile.Name() + ":/etc/envoy.yaml -v " +
			soPath + ":/etc/libgolang.so" +
			" -v /tmp:/tmp" +
			" -e GOCOVERDIR=" + coverDir +
			" " + strings.Join(envs, " ") +
			" -p " + dataPlanePort + ":" + dataPlanePort +
			" -p " + adminAPIPort + ":" + adminAPIPort + " " +
			hostAddr + " " +
			image

		envoyCmd = "envoy -c /etc/envoy.yaml"
	}

	// show why the configuration is invalid
	envoyValidateCmd := envoyCmd + " --mode validate -l error"

	content, _ := os.ReadFile(cfgFile.Name())
	digest := md5.Sum(content)
	if _, ok := validationCache[digest]; !ok {
		// Workaround for https://github.com/envoyproxy/envoy/issues/35961
		// TODO: drop this once we upgrade to Envoy 1.30+
		cfgFile, _ := os.Create(cfgFile.Name())
		opt.Bootstrap.WriteToForValidation(cfgFile)

		validateCmd := cmdline + " " + envoyValidateCmd
		cmds := strings.Fields(validateCmd)
		logger.Info("run validate cmd", "cmdline", validateCmd)

		cmd := exec.Command(cmds[0], cmds[1:]...)
		if isBinaryMode() {
			addEnvironemntVariables(cmd, opt.Envs)
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Info("bad envoy bootstrap configuration", "cmd", validateCmd, "output", string(out))
			return nil, err
		}

		validationCache[digest] = struct{}{}

		cfgFile, _ = os.Create(cfgFile.Name())
		cfgFile.Write(content)
	}

	if opt.LogLevel != "" {
		envoyCmd += " -l " + opt.LogLevel
	}

	cmdline = cmdline + " " + envoyCmd

	logger.Info("run cmd", "cmdline", cmdline)

	cmds := strings.Fields(cmdline)
	cmd := exec.Command(cmds[0], cmds[1:]...)
	if isBinaryMode() {
		addEnvironemntVariables(cmd, opt.Envs)
	}

	if isBinaryMode() {
		// Like the standard mode, we use stdout file to store the log of Envoy
		stdout, err := os.Create(filepath.Join(dir, "stdout"))
		if err != nil {
			return nil, err
		}

		// wrap writer to ensure the stderr and stdout won't affect each other
		w := &serializableFileWriter{File: stdout}
		cmd.Stdout = w

		// We don't need stderr file, which is used to store docker output in the standard mode.
		// Just left an empty file here to keep the same structure.
		_, err = os.Create(filepath.Join(dir, "stderr"))
		if err != nil {
			return nil, err
		}
		cmd.Stderr = w
	} else {
		stdout, err := os.Create(filepath.Join(dir, "stdout"))
		if err != nil {
			return nil, err
		}
		cmd.Stdout = stdout

		stderr, err := os.Create(filepath.Join(dir, "stderr"))
		if err != nil {
			return nil, err
		}
		cmd.Stderr = stderr
	}

	dp.cmd = cmd

	done := make(chan error)
	go func() {
		logger.Info("start envoy")
		err := dp.cmd.Start()
		if err != nil {
			logger.Error(err, "failed to start envoy")
			return
		}
		go func() { done <- cmd.Wait() }()
	}()

	if isBinaryMode() {
		// In binary mode, the port is open only after the control plane is up, which is called after
		// the data plane is up. So we don't check if the port is open. Instead, we wait for a while
		// to ensure the data plane can be started.
		waitTime := 1 * time.Second
		waitTimeEnv, _ := time.ParseDuration(os.Getenv("TEST_ENVOY_WAIT_BINARY_TO_START_TIME"))
		if waitTimeEnv != 0 {
			waitTime = waitTimeEnv
		}
		time.Sleep(waitTime)
	} else {
		helper.WaitServiceUp(t, ":"+dataPlanePort, "")
	}

	select {
	case err := <-done:
		return nil, err
	default:
	}

	dp.done = done

	return dp, nil
}

func (dp *DataPlane) root() string {
	pwd, _ := os.Getwd()
	name := dp.t.Name()
	dir := filepath.Join(pwd, testRootDirName, name)
	return dir
}

func (dp *DataPlane) cleanup(_ *testing.T) error {
	if !isBinaryMode() {
		cmd := exec.Command("docker", "stop", containerName)
		// ignore error when the containerName is not left over
		_ = cmd.Run()
	}

	dir := dp.root()
	_, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}

	// now the dir is not exist
	return nil
}

func (dp *DataPlane) Stop() {
	err := dp.FlushCoverage()
	if err != nil {
		logger.Error(err, "failed to flush coverage")
	}

	logger.Info("stop envoy")

	if isBinaryMode() {
		dp.cmd.Process.Signal(syscall.SIGTERM)
	} else {
		cmd := exec.Command("docker", "stop", containerName)
		err = cmd.Run()
	}
	if err != nil {
		logger.Error(err, "failed to terminate envoy")
		return
	}

	// ensure envoy is gone
	<-dp.done
	logger.Info("envoy stopped")

	var f *os.File
	if !isBinaryMode() {
		f = dp.cmd.Stderr.(*os.File)
		f.Close()
		f = dp.cmd.Stdout.(*os.File)
	} else {
		f = dp.cmd.Stdout.(*serializableFileWriter).Unbox()
	}

	f.Seek(0, 0)
	text, err := io.ReadAll(f)
	defer f.Close()
	if err != nil {
		logger.Error(err, "failed to read envoy stdout")
		return
	}

	if !dp.opt.NoErrorLogCheck {
		sc := bufio.NewScanner(bytes.NewReader(text))
		for sc.Scan() {
			s := sc.Text()
			if strings.Contains(s, "[error]") || strings.Contains(s, "[critical]") {
				assert.Falsef(dp.t, true, "error/critical level log found: %s", s)
				break
			}
		}
	}

	for _, pattern := range dp.opt.ExpectLogPattern {
		re := regexp.MustCompile(pattern)
		if !re.Match(text) {
			assert.Falsef(dp.t, true, "log pattern %q not found", pattern)
		}
	}

	for _, pattern := range dp.opt.ExpectNoLogPattern {
		re := regexp.MustCompile(pattern)
		if re.Match(text) {
			assert.Falsef(dp.t, true, "log pattern %q found", pattern)
		}
	}
}

func (dp *DataPlane) Head(path string, header http.Header) (*http.Response, error) {
	return dp.do("HEAD", path, header, nil)
}

func (dp *DataPlane) Get(path string, header http.Header) (*http.Response, error) {
	return dp.do("GET", path, header, nil)
}

func (dp *DataPlane) Delete(path string, header http.Header) (*http.Response, error) {
	return dp.do("DELETE", path, header, nil)
}

func (dp *DataPlane) Post(path string, header http.Header, body io.Reader) (*http.Response, error) {
	return dp.do("POST", path, header, body)
}

func (dp *DataPlane) PostWithTrailer(path string, header http.Header, body io.Reader, trailer http.Header) (*http.Response, error) {
	return dp.doWithTrailer("POST", path, header, body, trailer)
}

func (dp *DataPlane) Put(path string, header http.Header, body io.Reader) (*http.Response, error) {
	return dp.do("PUT", path, header, body)
}

func (dp *DataPlane) Patch(path string, header http.Header, body io.Reader) (*http.Response, error) {
	return dp.do("PATCH", path, header, body)
}

func (dp *DataPlane) SendAndCancelRequest(path string, after time.Duration) error {
	conn, err := net.DialTimeout("tcp", ":"+dp.dataPlanePort, 1*time.Second)
	if err != nil {
		return err
	}

	defer conn.Close()
	for _, s := range []string{
		fmt.Sprintf("POST %s HTTP/1.1\r\n", path),
		"Host: localhost\r\n",
		"Content-Length: 10000\r\n",
		"\r\n",
	} {
		_, err = conn.Write([]byte(s))
		if err != nil {
			return err
		}
	}
	time.Sleep(after)

	return nil
}

func (dp *DataPlane) do(method string, path string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://localhost:"+dp.dataPlanePort+path, body)
	if err != nil {
		return nil, err
	}
	req.Header = header
	tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
		return net.DialTimeout("tcp", ":"+dp.dataPlanePort, 1*time.Second)
	}}

	client := &http.Client{Transport: tr,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	return resp, err
}

func (dp *DataPlane) GetAdmin(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", "http://localhost:"+dp.adminAPIPort+path, nil)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
			return net.DialTimeout("tcp", ":"+dp.adminAPIPort, 1*time.Second)
		},
	}

	client := &http.Client{Transport: tr,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	return resp, err
}

func (dp *DataPlane) doWithTrailer(method string, path string, header http.Header, body io.Reader, trailer http.Header) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://localhost:"+dp.dataPlanePort+path, body)
	if err != nil {
		return nil, err
	}
	req.Header = header
	req.Header.Add("TE", "trailers")
	req.Trailer = trailer
	req.TransferEncoding = []string{"chunked"}
	tr := &http.Transport{
		DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
			return net.DialTimeout("tcp", ":"+dp.dataPlanePort, 1*time.Second)
		},
	}

	client := &http.Client{Transport: tr,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	return resp, err
}

// Use grpcurl so that the caller can specify the proto file without building the Go code.
// TODO: we can rewrite this in Go.
func (dp *DataPlane) Grpcurl(importPath, protoFile, fullMethodName, req string) ([]byte, error) {
	cmd := exec.Command("grpcurl", "-v", "-format-error", "-import-path", importPath, "-proto", protoFile, "-plaintext", "-d", req, ":"+dp.dataPlanePort, fullMethodName)
	dp.t.Logf("run grpcurl command: %s", cmd.String())
	return cmd.CombinedOutput()
}

func (dp *DataPlane) Configured() bool {
	resp, err := dp.Head("/detect_if_the_rds_takes_effect", nil)
	if err != nil {
		return false
	}
	if resp.StatusCode != 200 {
		return false
	}
	name := resp.Header.Get("route-version")
	if name == dp.latestRouteVersion {
		return false
	}
	dp.latestRouteVersion = name
	return true
}

func (dp *DataPlane) FlushCoverage() error {
	resp, err := dp.Post("/flush_coverage", nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (dp *DataPlane) SetLogLevel(loggerName string, level string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://0.0.0.0:%s/logging?%s=%s", dp.adminAPIPort, loggerName, level), bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (dp *DataPlane) AdminAPIPort() int {
	p, _ := strconv.Atoi(dp.adminAPIPort)
	return p
}

func (dp *DataPlane) Port() int {
	p, _ := strconv.Atoi(dp.dataPlanePort)
	return p
}
