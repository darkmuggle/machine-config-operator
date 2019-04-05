package operator

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	configv1 "github.com/openshift/api/config/v1"
	cov1helpers "github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ctrlcommon "github.com/openshift/machine-config-operator/pkg/controller/common"
	"github.com/openshift/machine-config-operator/pkg/version"
)

// syncVersion handles reporting the version to the clusteroperator
func (optr *Operator) syncVersion() error {
	co, err := optr.fetchClusterOperator()
	if err != nil {
		return err
	}
	if co == nil {
		return nil
	}

	// keep the old version and progressing if we fail progressing
	if cov1helpers.IsStatusConditionTrue(co.Status.Conditions, configv1.OperatorProgressing) && cov1helpers.IsStatusConditionTrue(co.Status.Conditions, configv1.OperatorFailing) {
		return nil
	}

	co.Status.Versions = optr.vStore.GetAll()
	// TODO(runcom): abstract below with updateStatus
	optr.setMachineConfigPoolStatuses(&co.Status)
	_, err = optr.configClient.ConfigV1().ClusterOperators().UpdateStatus(co)
	return err
}

// syncAvailableStatus applies the new condition to the mco's ClusterOperator object.
func (optr *Operator) syncAvailableStatus() error {
	co, err := optr.fetchClusterOperator()
	if err != nil {
		return err
	}
	if co == nil {
		return nil
	}

	optrVersion, _ := optr.vStore.Get("operator")
	failing := cov1helpers.IsStatusConditionTrue(co.Status.Conditions, configv1.OperatorFailing)
	message := fmt.Sprintf("Cluster has deployed %s", optrVersion)

	available := configv1.ConditionTrue

	if failing {
		available = configv1.ConditionFalse
		message = fmt.Sprintf("Cluster not available for %s", optrVersion)
	}

	coStatus := configv1.ClusterOperatorStatusCondition{
		Type:    configv1.OperatorAvailable,
		Status:  available,
		Message: message,
	}

	return optr.updateStatus(co, coStatus)
}

// syncProgressingStatus applies the new condition to the mco's ClusterOperator object.
func (optr *Operator) syncProgressingStatus() error {
	co, err := optr.fetchClusterOperator()
	if err != nil {
		return err
	}
	if co == nil {
		return nil
	}

	var (
		optrVersion, _ = optr.vStore.Get("operator")
		coStatus       = configv1.ClusterOperatorStatusCondition{
			Type:    configv1.OperatorProgressing,
			Status:  configv1.ConditionFalse,
			Message: fmt.Sprintf("Cluster version is %s", optrVersion),
		}
	)
	if optr.vStore.Equal(co.Status.Versions) {
		if optr.inClusterBringup {
			coStatus.Message = fmt.Sprintf("Cluster is bootstrapping %s", optrVersion)
			coStatus.Status = configv1.ConditionTrue
		}
	} else {
		coStatus.Message = fmt.Sprintf("Working towards %s", optrVersion)
		coStatus.Status = configv1.ConditionTrue
	}

	return optr.updateStatus(co, coStatus)
}

func (optr *Operator) updateStatus(co *configv1.ClusterOperator, status configv1.ClusterOperatorStatusCondition) error {
	existingCondition := cov1helpers.FindStatusCondition(co.Status.Conditions, status.Type)
	if existingCondition.Status != status.Status {
		status.LastTransitionTime = metav1.Now()
	}
	cov1helpers.SetStatusCondition(&co.Status.Conditions, status)
	optr.setMachineConfigPoolStatuses(&co.Status)
	_, err := optr.configClient.ConfigV1().ClusterOperators().UpdateStatus(co)
	return err
}

// syncFailingStatus applies the new condition to the mco's ClusterOperator object.
func (optr *Operator) syncFailingStatus(ierr error) (err error) {
	co, err := optr.fetchClusterOperator()
	if err != nil {
		return err
	}
	if co == nil {
		return nil
	}

	optrVersion, _ := optr.vStore.Get("operator")
	failing := configv1.ConditionTrue
	var message, reason string
	if ierr == nil {
		failing = configv1.ConditionFalse
	} else {
		if optr.vStore.Equal(co.Status.Versions) {
			// syncing the state to exiting version.
			message = fmt.Sprintf("Failed to resync %s because: %v", optrVersion, ierr.Error())
		} else {
			message = fmt.Sprintf("Unable to apply %s: %v", optrVersion, ierr.Error())
		}
		reason = ierr.Error()

		// set progressing
		if cov1helpers.IsStatusConditionTrue(co.Status.Conditions, configv1.OperatorProgressing) {
			cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorProgressing, Status: configv1.ConditionTrue, Message: fmt.Sprintf("Unable to apply %s", optrVersion)})
		} else {
			cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorProgressing, Status: configv1.ConditionFalse, Message: fmt.Sprintf("Error while reconciling %s", optrVersion)})
		}
	}

	coStatus := configv1.ClusterOperatorStatusCondition{
		Type: configv1.OperatorFailing,
		Status: failing,
		Message: message,
		Reason:  reason,
	}

	return optr.updateStatus(co, coStatus)
}

func (optr *Operator) fetchClusterOperator() (*configv1.ClusterOperator, error) {
	co, err := optr.configClient.ConfigV1().ClusterOperators().Get(optr.name, metav1.GetOptions{})
	if meta.IsNoMatchError(err) {
		return nil, nil
	}
	if apierrors.IsNotFound(err) {
		return optr.initializeClusterOperator()
	}
	if err != nil {
		return nil, err
	}
	return co, nil
}

func (optr *Operator) initializeClusterOperator() (*configv1.ClusterOperator, error) {
	co, err := optr.configClient.ConfigV1().ClusterOperators().Create(&configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: optr.name,
		},
	})
	if err != nil {
		return nil, err
	}
	cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorAvailable, Status: configv1.ConditionFalse})
	cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorProgressing, Status: configv1.ConditionFalse})
	cov1helpers.SetStatusCondition(&co.Status.Conditions, configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorFailing, Status: configv1.ConditionFalse})
	// RelatedObjects are consumed by https://github.com/openshift/must-gather
	co.Status.RelatedObjects = []configv1.ObjectReference{
		{Resource: "namespaces", Name: "openshift-machine-config-operator"},
	}
	// During an installation we report the RELEASE_VERSION as soon as the component is created
	// whether for normal runs and upgrades this code isn't hit and we get the right version every
	// time. This also only contains the operator RELEASE_VERSION when we're here.
	co.Status.Versions = optr.vStore.GetAll()
	return optr.configClient.ConfigV1().ClusterOperators().UpdateStatus(co)
}

func (optr *Operator) setMachineConfigPoolStatuses(status *configv1.ClusterOperatorStatus) {
	statuses, err := optr.allMachineConfigPoolStatus()
	if err != nil {
		glog.Error(err)
		return
	}
	raw, err := json.Marshal(statuses)
	if err != nil {
		glog.Error(err)
		return
	}
	status.Extension.Raw = raw
}

func (optr *Operator) allMachineConfigPoolStatus() (map[string]string, error) {
	pools, err := optr.mcpLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	ret := map[string]string{}
	for _, pool := range pools {
		p := pool.DeepCopy()
		err := isMachineConfigPoolConfigurationValid(p, version.Version.String(), optr.mcLister.Get)
		if err != nil {
			glog.V(4).Infof("Skipping status for pool %s because %v", p.GetName(), err)
			continue
		}
		ret[p.GetName()] = machineConfigPoolStatus(p)
	}
	return ret, nil
}

// isMachineConfigPoolConfigurationValid returns nil error when the configuration of a `pool` is created by the controller at version `version`.
func isMachineConfigPoolConfigurationValid(pool *mcfgv1.MachineConfigPool, version string, machineConfigGetter func(string) (*mcfgv1.MachineConfig, error)) error {
	// both .status.configuration.name and .status.configuration.source must be set.
	if len(pool.Status.Configuration.Name) == 0 {
		return fmt.Errorf("configuration for pool %s is empty", pool.GetName())
	}
	if len(pool.Status.Configuration.Source) == 0 {
		return fmt.Errorf("list of MachineConfigs that were used to generate configuration for pool %s is empty", pool.GetName())
	}
	mcs := []string{pool.Status.Configuration.Name}
	for _, fragment := range pool.Status.Configuration.Source {
		mcs = append(mcs, fragment.Name)
	}
	for _, mcName := range mcs {
		mc, err := machineConfigGetter(mcName)
		if err != nil {
			return err
		}

		// We check that all the machineconfigs (generated, and those that were used to create generated,
		// not the user provided ones that do not have a version) were generated by correct version of the controller.
		v, ok := mc.Annotations[ctrlcommon.GeneratedByControllerVersionAnnotationKey]
		// The generated machineconfig from fragments for the pool MUST have a version and the annotation.
		// The bootstrapped MCs fragments do have this annotation but we don't fail (???) if they don't have
		// the annotation for some reason.
		if !ok && pool.Status.Configuration.Name == mcName {
			return fmt.Errorf("%s must be created by controller version %s", mcName, version)
		}
		// user provided MC fragments do not have the annotation, so we just skip version check there
		// The check below is just for: 1) the generated MC for the pool, 2) the bootstrapped fragments
		// that do have this annotation with a version.
		if ok && v != version {
			return fmt.Errorf("controller version mismatch for %s expected %s has %s", mcName, version, v)
		}
	}
	return nil
}

func machineConfigPoolStatus(pool *mcfgv1.MachineConfigPool) string {
	switch {
	case mcfgv1.IsMachineConfigPoolConditionTrue(pool.Status.Conditions, mcfgv1.MachineConfigPoolUpdated):
		return fmt.Sprintf("all %d nodes are at latest configuration %s", pool.Status.MachineCount, pool.Status.Configuration.Name)
	case mcfgv1.IsMachineConfigPoolConditionTrue(pool.Status.Conditions, mcfgv1.MachineConfigPoolUpdating):
		return fmt.Sprintf("%d out of %d nodes have updated to latest configuration %s", pool.Status.UpdatedMachineCount, pool.Status.MachineCount, pool.Status.Configuration.Name)
	default:
		return "<unknown>"
	}
}
