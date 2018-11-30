package dummy

import (
	"testing"
	"time"

	plug "github.com/openshift/machine-config-operator/pkg/daemon/internal/plugins"
)

func TestDummy(t *testing.T) {
	t.Log("testing dummy plugin registration")

	stopCh := make(chan struct{})
	defer close(stopCh)

	plug.ExecPlugins(stopCh)
	time.Sleep(2 * time.Second)

}
