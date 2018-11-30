package internal

// Pluginer describes required functions of plugins.
type Pluginer interface {
	Name() string                    // Returns name of the plugin.
	Kind() string                    // Returns plugin or daemon.
	Exec(stop <-chan struct{}) error // executes the plugin
}
