package controller

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	"github.com/rossigee/provider-vault/internal/controller/leaserenewal"
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
	if err := setupRBAC(mgr.GetClient(), o.Logger); err != nil {
		o.Logger.Info("RBAC setup warning (may be transient)", "error", err)
	}
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
		leaserenewal.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

func setupRBAC(c client.Client, l logging.Logger) error {
	ctx := context.Background()

	rules := []rbacv1.PolicyRule{
		{APIGroups: []string{"vault.m.crossplane.io"}, Resources: []string{"providerconfigs", "providerconfigs/status", "providerconfigusages", "providerconfigusages/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"approlesecretid.vault.m.crossplane.io"}, Resources: []string{"approlesecretids", "approlesecretids/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"auditdevice.vault.m.crossplane.io"}, Resources: []string{"auditdevices", "auditdevices/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"authbackendrole.vault.m.crossplane.io"}, Resources: []string{"authbackendroles", "authbackendroles/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"authmethod.vault.m.crossplane.io"}, Resources: []string{"authmethods", "authmethods/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"awsauthconfig.vault.m.crossplane.io"}, Resources: []string{"awsauthconfigs", "awsauthconfigs/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"azureauthconfig.vault.m.crossplane.io"}, Resources: []string{"azureauthconfigs", "azureauthconfigs/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"certificate.vault.m.crossplane.io"}, Resources: []string{"certificates", "certificates/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"databasebackend.vault.m.crossplane.io"}, Resources: []string{"databasebackends", "databasebackends/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"databaserole.vault.m.crossplane.io"}, Resources: []string{"databaseroles", "databaseroles/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"gcpauthconfig.vault.m.crossplane.io"}, Resources: []string{"gcpauthconfigs", "gcpauthconfigs/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"identityentity.vault.m.crossplane.io"}, Resources: []string{"identityentities", "identityentities/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"identitygroup.vault.m.crossplane.io"}, Resources: []string{"identitygroups", "identitygroups/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"jwtauthconfig.vault.m.crossplane.io"}, Resources: []string{"jwtauthconfigs", "jwtauthconfigs/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"kvsecret.vault.m.crossplane.io"}, Resources: []string{"kvsecrets", "kvsecrets/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"kubernetesauthconfig.vault.m.crossplane.io"}, Resources: []string{"kubernetesauthconfigs", "kubernetesauthconfigs/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"ldapauthconfig.vault.m.crossplane.io"}, Resources: []string{"ldapauthconfigs", "ldapauthconfigs/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"mount.vault.m.crossplane.io"}, Resources: []string{"mounts", "mounts/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"namespaces.vault.m.crossplane.io"}, Resources: []string{"namespaces", "namespaces/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"pkiconfig.vault.m.crossplane.io"}, Resources: []string{"pkiconfigs", "pkiconfigs/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"policy.vault.m.crossplane.io"}, Resources: []string{"policies", "policies/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"quota.vault.m.crossplane.io"}, Resources: []string{"quotas", "quotas/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"secretbackendrole.vault.m.crossplane.io"}, Resources: []string{"secretbackendroles", "secretbackendroles/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"token.vault.m.crossplane.io"}, Resources: []string{"tokens", "tokens/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"transitkey.vault.m.crossplane.io"}, Resources: []string{"transitkeys", "transitkeys/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{APIGroups: []string{"leaserenewal.vault.m.crossplane.io"}, Resources: []string{"leaserenewals", "leaserenewals/status"}, Verbs: []string{"get", "list", "watch", "update", "patch", "create"}},
		{
			APIGroups: []string{"vault.m.crossplane.io", "approlesecretid.vault.m.crossplane.io", "auditdevice.vault.m.crossplane.io", "authbackendrole.vault.m.crossplane.io", "authmethod.vault.m.crossplane.io", "awsauthconfig.vault.m.crossplane.io", "azureauthconfig.vault.m.crossplane.io", "certificate.vault.m.crossplane.io", "databasebackend.vault.m.crossplane.io", "databaserole.vault.m.crossplane.io", "gcpauthconfig.vault.m.crossplane.io", "identityentity.vault.m.crossplane.io", "identitygroup.vault.m.crossplane.io", "jwtauthconfig.vault.m.crossplane.io", "kvsecret.vault.m.crossplane.io", "kubernetesauthconfig.vault.m.crossplane.io", "ldapauthconfig.vault.m.crossplane.io", "mount.vault.m.crossplane.io", "namespaces.vault.m.crossplane.io", "pkiconfig.vault.m.crossplane.io", "policy.vault.m.crossplane.io", "quota.vault.m.crossplane.io", "secretbackendrole.vault.m.crossplane.io", "token.vault.m.crossplane.io", "transitkey.vault.m.crossplane.io", "leaserenewal.vault.m.crossplane.io"},
			Resources: []string{"*/finalizers"},
			Verbs:     []string{"update"},
		},
		{APIGroups: []string{"", "coordination.k8s.io"}, Resources: []string{"secrets", "configmaps", "events", "leases"}, Verbs: []string{"*"}},
	}

	system := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "crossplane:provider:provider-vault:system",
			Labels: map[string]string{"rbac.crossplane.io/system": "provider-vault"},
		},
		Rules: rules,
	}
	if err := c.Create(ctx, system); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	if err := c.Update(ctx, system); err != nil {
		l.Info("system role update", "err", err)
	}

	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "crossplane:provider:provider-vault:system"},
		RoleRef: rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "crossplane:provider:provider-vault:system"},
		Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "provider-vault", Namespace: "crossplane-system"}},
	}
	if err := c.Create(ctx, binding); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	if err := c.Update(ctx, binding); err != nil {
		l.Info("system binding update", "err", err)
	}

	edit := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "crossplane:provider:provider-vault:aggregate-to-edit",
			Labels: map[string]string{
				"rbac.crossplane.io/aggregate-to-edit": "true", "rbac.crossplane.io/aggregate-to-admin": "true",
				"rbac.crossplane.io/aggregate-to-crossplane": "true", "rbac.crossplane.io/system": "provider-vault",
			},
		},
		Rules: withVerbs(rules, []string{"*"}),
	}
	if err := c.Create(ctx, edit); err != nil && !errors.IsAlreadyExists(err) {
		l.Info("aggregate-to-edit create warning (non-fatal)", "err", err)
	}
	_ = c.Update(ctx, edit)

	view := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "crossplane:provider:provider-vault:aggregate-to-view",
			Labels: map[string]string{"rbac.crossplane.io/aggregate-to-view": "true", "rbac.crossplane.io/system": "provider-vault"},
		},
		Rules: withVerbs(rules, []string{"get", "list", "watch"}),
	}
	if err := c.Create(ctx, view); err != nil && !errors.IsAlreadyExists(err) {
		l.Info("aggregate-to-view create warning (non-fatal)", "err", err)
	}
	_ = c.Update(ctx, view)

	l.Info("provider self-managed RBAC roles ensured")
	return nil
}

func withVerbs(r []rbacv1.PolicyRule, verbs []string) []rbacv1.PolicyRule {
	out := make([]rbacv1.PolicyRule, len(r))
	for i := range r {
		out[i] = r[i]
		out[i].Verbs = verbs
	}
	return out
}
