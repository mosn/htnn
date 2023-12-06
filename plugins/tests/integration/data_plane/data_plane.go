package data_plane

import (
	"bufio"
	"context"
	"crypto/md5"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/pkg/log"
	"mosn.io/moe/plugins/tests/integration/helper"
)

var (
	logger = log.DefaultLogger.WithName("data_plane")

	testRootDirName = "test-envoy"
	containerName   = "run_envoy_for_test"

	validationCache = map[[16]byte]struct{}{}
)

type DataPlane struct {
	cmd  *exec.Cmd
	t    *testing.T
	opt  *Option
	done chan error
}

type Option struct {
	LogLevel      string
	CheckErrorLog bool
}

func StartDataPlane(t *testing.T, opt *Option) (*DataPlane, error) {
	if opt == nil {
		opt = &Option{
			LogLevel:      "info",
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

	cfgFilename := "envoy.yaml"
	cfgFile, err := os.Create(filepath.Join(dir, cfgFilename))
	if err != nil {
		return nil, err
	}

	err = WriteBoostrapConfig(cfgFile)
	cfgFile.Close()
	if err != nil {
		return nil, err
	}

	envoyCmd := "envoy -c /etc/envoy.yaml"
	envoyValidateCmd := envoyCmd + " --mode validate -l critical"
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

	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}

	networkName := "ci_service"
	err = exec.Command("docker", "network", "inspect", networkName).Run()
	if err != nil {
		logger.Info("docker network used by test not found, create one")
		err = exec.Command("docker", "network", "create", networkName).Run()
		if err != nil {
			return nil, err
		}
	}

	// This is the envoyproxy/envoy:contrib-debug-dev fetched in 2023-11-22
	// Use docker inspect --format='{{index .RepoDigests 0}}' envoyproxy/envoy:contrib-debug-dev
	// to get the sha256 ID
	image := "envoyproxy/envoy@sha256:1fa13772ad01292fdbd73541717ef1a65fcdb2350bf13c173bddb10bf1f36c7c"
	pwd, _ := os.Getwd()
	projectRoot := filepath.Join(pwd, "..", "..", "..")
	cmdline := "docker run" +
		" --name " + containerName +
		" --network " + networkName +
		" --user " + currentUser.Uid +
		" --rm -t -v " +
		cfgFile.Name() + ":/etc/envoy.yaml -v " +
		projectRoot +
		"/plugins/tests/integration/libgolang.so:/etc/libgolang.so" +
		" -v /tmp:/tmp" +
		" -p 10000:10000 -p 9998:9998 " + hostAddr + " " +
		image

	content, _ := os.ReadFile(cfgFile.Name())
	digest := md5.Sum(content)
	if _, ok := validationCache[digest]; !ok {
		validateCmd := cmdline + " " + envoyValidateCmd
		cmds := strings.Fields(validateCmd)
		out, err := exec.Command(cmds[0], cmds[1:]...).CombinedOutput()
		if err != nil {
			logger.Info("bad envoy bootstrap configuration", "output", string(out))
			return nil, err
		}

		validationCache[digest] = struct{}{}
	}

	cmdline = cmdline + " " + envoyCmd

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

	helper.WaitServiceUp(t, ":10000", "failed to start data plane")

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
	<-dp.done
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

func (dp *DataPlane) Put(path string, header http.Header, body io.Reader) (*http.Response, error) {
	return dp.do("PUT", path, header, body)
}

func (dp *DataPlane) Patch(path string, header http.Header, body io.Reader) (*http.Response, error) {
	return dp.do("PATCH", path, header, body)
}

func (dp *DataPlane) do(method string, path string, header http.Header, body io.Reader) (*http.Response, error) {
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

func (dp *DataPlane) Configured() bool {
	_, err := dp.Head("/echo?detect_if_the_rds_takes_effect", nil)
	if err == nil {
		return true
	}
	return false
}
