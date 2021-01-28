package daemon

import (
	"io/ioutil"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// NotCoreOSClient is a wrapper around RpmOstreeClient that implements NodeUpdaterClient
// safely on unsupported and legacy Operating systems.
type notCoreOSClient struct {
}

// NewNodeUpdaterClientNotCoreOS returns a NodeUpdaterClient for legacy and non-CoreOS
// operating systems like RHEL 7.
func NewNodeUpdaterClientNotCoreOS() NodeUpdaterClient {
	glog.Warning("Operating System is not a CoreOS Variant. Update functionality is disabled.")
	return &notCoreOSClient{}
}

var (
	// notCoreOSClient is a NodeUpdaterClient.
	_ NodeUpdaterClient = &notCoreOSClient{}

	// ErrNotCoreosVariant is a generic error for un-supported tasks on legacy systems.
	ErrNotCoreosVariant = errors.New("operating system is not a CoreOS Variant")
)

// GetBootedDeployment returns the booted demployment.
func (noCos *notCoreOSClient) GetBootedDeployment() (*RpmOstreeDeployment, error) {
	return &RpmOstreeDeployment{}, nil
}

// Rebase calls rpm-ostree status
func (noCos *notCoreOSClient) Rebase(a, b string) (bool, error) {
	glog.Info("Rebase is not supported on this system.")
	return false, ErrNotCoreosVariant
}

// GetStatus returns the rpm-ostree status
func (noCos *notCoreOSClient) GetStatus() (string, error) {
	return "", ErrNotCoreosVariant
}

// GetBootedOSImageURL returns the rpmOstree Booted OS Image
func (noCos *notCoreOSClient) GetBootedOSImageURL() (string, string, error) {
	return "", "", ErrNotCoreosVariant
}

// GetKernelArgs returns the real kernel arguments since we can't use rpmOstree
func (noCos *notCoreOSClient) GetKernelArgs() ([]string, error) {
	content, err := ioutil.ReadFile(CmdLineFile)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), " "), nil
}

// SetKernelArgs sets the kernel arguments
func (noCos *notCoreOSClient) SetKernelArgs([]KernelArgument) (string, error) {
	return "unsupported", ErrNotCoreosVariant
}

// RemovePendingDeployment is not supported on non-CoreOS machines.
func (noCos *notCoreOSClient) RemovePendingDeployment() error {
	return ErrNotCoreosVariant
}

func (noCos *notCoreOSClient) RunRpmOstree(noun string, args ...string) ([]byte, error) {
	return nil, ErrNotCoreosVariant
}
