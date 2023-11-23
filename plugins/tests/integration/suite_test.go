package plugins

import (
	"os"
	"testing"
	"time"

	"mosn.io/moe/plugins/tests/integration/control_plane"
	_ "mosn.io/moe/plugins/tests/pkg/envoy"
)

var (
	controlPlane *control_plane.ControlPlane
)

func TestMain(m *testing.M) {
	controlPlane = control_plane.NewControlPlane()
	go func() {
		controlPlane.Start()
	}()
	time.Sleep(1 * time.Second)

	os.Exit(m.Run())
}
