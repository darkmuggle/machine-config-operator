package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/golang/glog"
)

// systemdInhibit executes systemd-inhibit by running an infinite sleep command.
// The inhibit will exist until either the exitCh recieves an error or
// the command is cancelled. The Kublet uses a similiar mechanism, although we
// can't use it here since the Kublet uses a timer to set the inhbitor.
func systemdInhibit() (func(), error) {
	modes := []string{
		"shutdown",
		"sleep",
		"idle",
		"handle-power-key",
		"handle-suspend-key",
		"handle-hibernate-key",
		"handle-lid-switch",
	}

	glog.Info("Inhibiting power states changes via systemd")

	args := []string{
		fmt.Sprintf("--what='%v'", strings.Join(modes, ":")),
		fmt.Sprintf("--who='MCD Pid %d", os.Getpid()),
		"--why='Update Operation'",
		"/bin/sleep", "inifity",
	}
	cmd := exec.Command("systemd-inhibit", args...)

	// done terminates the inhibitor
	done := func() {
		if !cmd.ProcessState.Exited() {
			glog.Info("Releasing systemd inhibitor")
			_ = cmd.Process.Kill()
		}
		glog.Info("Released systemd inhibitor")
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	err := cmd.Start()

	// watch the inhibitor and make sure its cleaned up.
	go func() {
		for {
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				break
			}
			select {
			case <-sigs:
				done()
			}
		}
	}()

	return done, err
}
