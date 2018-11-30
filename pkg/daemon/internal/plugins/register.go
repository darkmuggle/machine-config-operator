package plugin

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	dIn "github.com/openshift/machine-config-operator/pkg/daemon/internal"
)

// plugins hold the registered plugins.
var plugins = []dIn.Pluginer{}

// Register appends Plugins to the plugin list. Plugins should call Register in
// its init function. See plugin/dummy/dummy.go for an exmaple.
func Register(p dIn.Pluginer) {
	plugins = append(plugins, p)
	glog.Info("plugin registered: %s as type %s", p.Name(), p.Kind())
}

// ExecPlugins executes all registered plugins.
// TODO: this is a POC, and obviously this is rather sophomoric, so make it better.
func ExecPlugins(stop <-chan struct{}) {
	perrs := make(chan error, len(plugins))

	var wg sync.WaitGroup
	for _, p := range plugins {
		go func() {
			wg.Add(1)
			defer wg.Done()
			perrs <- p.Exec(stop)
		}()
	}

	go func() {
		wg.Wait()
		fmt.Printf("%v", len(perrs))

		for e := range perrs {
			glog.Error("plugin:", e)
		}
	}()

}
