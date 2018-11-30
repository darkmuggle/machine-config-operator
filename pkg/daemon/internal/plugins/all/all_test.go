package all

// Import the plugins that should be run during the test run.

import (
	// import the dummy plugin
	_ "github.com/openshift/machine-config-operator/pkg/daemon/internal/plugins/dummy"
)
