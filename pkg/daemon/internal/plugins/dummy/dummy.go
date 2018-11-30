package dummy

// Dummy is an no-op example plugin. Its entire purpouse is to _demonstrate_
// the idea and basic structure of a plugin.

import (
	"fmt"

	dIn "github.com/openshift/machine-config-operator/pkg/daemon/internal"
	plug "github.com/openshift/machine-config-operator/pkg/daemon/internal/plugins"
)

const (
	defPlugName = "dummy"
	defPlugType = "plugin"
)

var (
	plugName   = defPlugName
	plugErrEnd = fmt.Errorf("plugin %v: recieved stop signal", plugName)
)

// Dummy is a basic plugin.
type Dummy struct{}

// Dummy implements the Plugin interface.
var _ dIn.Pluginer = (*Dummy)(nil)

// Init registers the dummy plugin.
func init() {
	var d Dummy
	plug.Register(&d)
}

// Name returns the name of the plugin.
func (p *Dummy) Name() string {
	return plugName
}

// Kind returns the kind of plugin
func (p *Dummy) Kind() string {
	return defPlugType
}

// Exec handles the execution of the plugin. Since this is a dummy plugin,
// it does nothing.
func (p *Dummy) Exec(stop <-chan struct{}) error {
	<-stop
	return plugErrEnd
}
