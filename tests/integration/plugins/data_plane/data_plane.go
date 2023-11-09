package data_plane

import (
	"bufio"
	"context"
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

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("data_plane")

	testRootDirName = "test-envoy"
	containerName   = "run_envoy_for_test"
)

type DataPlane struct {
	cmd *exec.Cmd
	t   *testing.T
	opt *Option
}

type Option struct {
	LogLevel      string
	CheckErrorLog bool
}

func StartDataPlane(t *testing.T, opt *Option) (*DataPlane, error) {
	if opt == nil {
		opt = &Option{
			LogLevel:      "debug",
			CheckErrorLog: true,
		}
	}

	dp := &DataPlane{
		t:   t,
		opt: opt,
	}
	err := dp.cleanup(t)
	if err != nil {
		return nil, err
	}

	dir := dp.root()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	envoyCmd := "envoy -c /etc/envoy.yaml"
	if opt.LogLevel != "" {
		envoyCmd += " -l " + opt.LogLevel
	}

	hostAddr := ""
	if runtime.GOOS == "linux" {
		// We use this special domain to access the control plane on host.
		// It works with Docker for Win/Mac (--network host doesn't work).
		// For Linux's Docker, a special option is used instead
		hostAddr = "--add-host=host.docker.internal:host-gateway"
	}

	// This is the envoyproxy/envoy:contrib-debug-dev fetched in 2023-10-27
	// Use docker inspect --format='{{index .RepoDigests 0}}' envoyproxy/envoy:contrib-debug-dev
	// to get the sha256 ID
	image := "envoyproxy/envoy@sha256:216c1c849c326ffad0a249de6886fd90c1364bbac193d5b1e36846098615071b"
	pwd, _ := os.Getwd()
	projectRoot := filepath.Join(pwd, "..", "..", "..")
	cmdline := "docker run --name " +
		containerName + " --rm -t -v " +
		projectRoot +
		"/tests/integration/plugins/data_plane/envoy.yaml:/etc/envoy.yaml -v " +
		projectRoot +
		"/tests/integration/plugins/libgolang.so:/etc/libgolang.so" +
		" -p 10000:10000 -p 9998:9998 " + hostAddr + " " +
		image + " " + envoyCmd

	logger.Info("run cmd", "cmdline", cmdline)

	cmds := strings.Fields(cmdline)
	cmd := exec.Command(cmds[0], cmds[1:]...)

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
	dp.cmd = cmd

	go func() {
		logger.Info("start envoy")
		err := dp.cmd.Start()
		if err != nil {
			logger.Error(err, "failed to start envoy")
		}
	}()
	time.Sleep(5 * time.Second)

	return dp, nil
}

func (dp *DataPlane) root() string {
	pwd, _ := os.Getwd()
	projectRoot := filepath.Join(pwd, "..", "..", "..")
	name := dp.t.Name()
	dir := filepath.Join(projectRoot, testRootDirName, name)
	return dir
}

func (dp *DataPlane) cleanup(t *testing.T) error {
	cmd := exec.Command("docker", "stop", containerName)
	// ignore error when the containerName is not left over
	_ = cmd.Run()

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
	logger.Info("stop envoy")
	cmd := exec.Command("docker", "stop", containerName)
	err := cmd.Run()
	if err != nil {
		logger.Error(err, "failed to terminate envoy")
		return
	}

	// ensure envoy is gone
	_ = dp.cmd.Wait()
	logger.Info("envoy stopped")

	f := dp.cmd.Stdout.(*os.File)
	if dp.opt.CheckErrorLog {
		f.Seek(0, 0)
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			s := sc.Text()
			if strings.Contains(s, "[error]") || strings.Contains(s, "[critical]") {
				assert.Falsef(dp.t, true, "error/critical level log found: %s", s)
				break
			}
		}
	}

	f.Close()
	f = dp.cmd.Stderr.(*os.File)
	f.Close()
}

func (dp *DataPlane) Get(path string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest("GET", "http://localhost:10000"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header = header
	return dp.do(req)
}

func (dp *DataPlane) Post(path string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", "http://localhost:10000"+path, body)
	if err != nil {
		return nil, err
	}
	req.Header = header
	return dp.do(req)
}

func (dp *DataPlane) do(req *http.Request) (*http.Response, error) {
	tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
		return net.DialTimeout("tcp", ":10000", 1*time.Second)
	}}

	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	return resp, err
}
