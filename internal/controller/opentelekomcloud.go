package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/peertechde/provider-opentelekomcloud/internal/controller/config"
	"github.com/peertechde/provider-opentelekomcloud/internal/controller/securitygroup"
	"github.com/peertechde/provider-opentelekomcloud/internal/controller/securitygrouprule"
	"github.com/peertechde/provider-opentelekomcloud/internal/controller/subnet"
	"github.com/peertechde/provider-opentelekomcloud/internal/controller/vpc"
)

// SetupGated creates all OpenTelekomCloud controllers with safe-start support and adds them to
// the supplied manager.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		vpc.SetupGated,
		subnet.SetupGated,
		securitygroup.SetupGated,
		securitygrouprule.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
