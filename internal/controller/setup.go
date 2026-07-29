package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rossigee/provider-vault/internal/controller/authbackendrole"
	"github.com/rossigee/provider-vault/internal/controller/authmethod"
	"github.com/rossigee/provider-vault/internal/controller/certificate"
	"github.com/rossigee/provider-vault/internal/controller/kvsecret"
	"github.com/rossigee/provider-vault/internal/controller/mount"
	"github.com/rossigee/provider-vault/internal/controller/pkiconfig"
	"github.com/rossigee/provider-vault/internal/controller/policy"
	"github.com/rossigee/provider-vault/internal/controller/secretbackendrole"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		kvsecret.Setup,
		policy.Setup,
		authmethod.Setup,
		mount.Setup,
		secretbackendrole.Setup,
		authbackendrole.Setup,
		pkiconfig.Setup,
		certificate.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
