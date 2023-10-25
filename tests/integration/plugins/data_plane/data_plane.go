package data_plane

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
}

func StartDataPlane(t *testing.T) (*DataPlane, error) {
	dp := &DataPlane{
		t: t,
	}
	err := dp.cleanup(t)
	if err != nil {
		return nil, err
	}

	dir := dp.root()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	image := "envoyproxy/envoy:contrib-debug-dev"
	pwd, _ := os.Getwd()
	projectRoot := filepath.Join(pwd, "../../..")
	cmdline := "docker run --name " +
		containerName + " --rm -t -v " +
		projectRoot +
		"/tests/integration/plugins/data_plane/envoy.yaml:/etc/envoy.yaml -v " +
		projectRoot +
		"/libgolang.so:/etc/libgolang.so" + " -p 10000:10000 -p 9998:9998 " +
		image + " envoy -c /etc/envoy.yaml"

	logger.Info("run cmd", "cmdline", cmdline)

	cmds := strings.Split(cmdline, " ")
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
		dp.cmd.Start()
	}()
	time.Sleep(5 * time.Second)

	return dp, nil
}

func (dp *DataPlane) root() string {
	pwd, _ := os.Getwd()
	projectRoot := filepath.Join(pwd, "../../..")
	name := dp.t.Name()
	dir := filepath.Join(projectRoot, testRootDirName, name)
	return dir
}

func (dp *DataPlane) cleanup(t *testing.T) error {
	cmd := exec.Command("docker", "stop", containerName)
	// ignore error when the containerName is not left over
	cmd.Run()

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
	dp.cmd.Wait()
	logger.Info("envoy stopped")

	f := dp.cmd.Stdout.(*os.File)
	f.Close()
	f = dp.cmd.Stderr.(*os.File)
	f.Close()
}

func (dp *DataPlane) Do(method string, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://localhost:10000"+path, body)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
		return net.DialTimeout("tcp", ":10000", 1*time.Second)
	}}

	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	return resp, err
}
