package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/rossigee/provider-vault/internal/controller/approlesecretid"
	"github.com/rossigee/provider-vault/internal/controller/auditdevice"
	"github.com/rossigee/provider-vault/internal/controller/authbackendrole"
	"github.com/rossigee/provider-vault/internal/controller/authmethod"
	"github.com/rossigee/provider-vault/internal/controller/awsauthconfig"
	"github.com/rossigee/provider-vault/internal/controller/azureauthconfig"
	"github.com/rossigee/provider-vault/internal/controller/gcpauthconfig"
	"github.com/rossigee/provider-vault/internal/controller/certificate"
	"github.com/rossigee/provider-vault/internal/controller/databasebackend"
	"github.com/rossigee/provider-vault/internal/controller/databaserole"
	"github.com/rossigee/provider-vault/internal/controller/identityentity"
	"github.com/rossigee/provider-vault/internal/controller/identitygroup"
	"github.com/rossigee/provider-vault/internal/controller/kubernetesauthconfig"
	"github.com/rossigee/provider-vault/internal/controller/jwtauthconfig"
	"github.com/rossigee/provider-vault/internal/controller/kvsecret"
	"github.com/rossigee/provider-vault/internal/controller/ldapauthconfig"
	"github.com/rossigee/provider-vault/internal/controller/mount"
	"github.com/rossigee/provider-vault/internal/controller/namespaces"
	"github.com/rossigee/provider-vault/internal/controller/pkiconfig"
	"github.com/rossigee/provider-vault/internal/controller/policy"
	"github.com/rossigee/provider-vault/internal/controller/quota"
	"github.com/rossigee/provider-vault/internal/controller/secretbackendrole"
	"github.com/rossigee/provider-vault/internal/controller/token"
	"github.com/rossigee/provider-vault/internal/controller/transitkey"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		approlesecretid.Setup,
		auditdevice.Setup,
		awsauthconfig.Setup,
		azureauthconfig.Setup,
		gcpauthconfig.Setup,
		kvsecret.Setup,
		policy.Setup,
		authmethod.Setup,
		mount.Setup,
		secretbackendrole.Setup,
		authbackendrole.Setup,
		pkiconfig.Setup,
		certificate.Setup,
		databasebackend.Setup,
		databaserole.Setup,
		transitkey.Setup,
		identityentity.Setup,
		identitygroup.Setup,
		kubernetesauthconfig.Setup,
		jwtauthconfig.Setup,
		ldapauthconfig.Setup,
		token.Setup,
		quota.Setup,
		namespaces.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
