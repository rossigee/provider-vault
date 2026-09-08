package approlesecretid

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/rossigee/provider-vault/internal/features"

	v1beta1 "github.com/rossigee/provider-vault/apis/approlesecretid/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotAppRoleSecretID = "managed resource is not an AppRoleSecretID custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errCreateSecretID     = "cannot generate AppRole SecretID"
	errLookupSecretID     = "cannot lookup AppRole SecretID"
	errDestroySecretID    = "cannot destroy AppRole SecretID"
	errReadRoleID         = "cannot read AppRole role-id"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.AppRoleSecretIDKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true),
	}
	if o.Features.Enabled(features.EnableAlphaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.AppRoleSecretIDGroupVersionKind),
		opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.AppRoleSecretID{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.AppRoleSecretID)
	if !ok {
		return nil, errors.New(errNotAppRoleSecretID)
	}

	if err := clients.TrackUsage(ctx, c.kube, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.Connect(ctx, c.kube, cr)
	if err != nil {
		return nil, err
	}

	return &external{service: svc, kube: c.kube}, nil
}

type external struct {
	kube    client.Client
	service *clients.VaultClient
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.AppRoleSecretID)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAppRoleSecretID)
	}

	accessor := meta.GetExternalName(cr)
	if accessor == "" || accessor == cr.GetName() {
		_, accessor = e.getSecretID(ctx, cr)
	}
	if accessor == "" {
		accessor = cr.Status.AtProvider.SecretIDAccessor
	}
	if accessor == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	params := cr.Spec.ForProvider
	_, err := e.service.LookupAppRoleSecretID(ctx, params.Backend, params.RoleName, accessor)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	if cr.Status.AtProvider.RoleID == "" {
		if roleID, err := e.service.ReadAppRoleRoleID(ctx, params.Backend, params.RoleName); err == nil {
			cr.Status.AtProvider.RoleID = roleID
		}
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.AppRoleSecretID)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAppRoleSecretID)
	}

	params := cr.Spec.ForProvider
	vaultParams := map[string]interface{}{}
	if len(params.Metadata) > 0 {
		metaJSON, err := json.Marshal(params.Metadata)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, "cannot marshal metadata")
		}
		vaultParams["metadata"] = string(metaJSON)
	}
	if len(params.CidrList) > 0 {
		vaultParams["cidr_list"] = params.CidrList
	}
	if len(params.TokenBoundCIDRs) > 0 {
		vaultParams["token_bound_cidrs"] = params.TokenBoundCIDRs
	}

	data, err := e.service.GenerateAppRoleSecretID(ctx, params.Backend, params.RoleName, vaultParams)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateSecretID)
	}

	secretID, _ := data["secret_id"].(string)
	secretIDAccessor, _ := data["secret_id_accessor"].(string)

	meta.SetExternalName(cr, secretIDAccessor)

	connectionDetails := managed.ConnectionDetails{
		"secret_id":          []byte(secretID),
		"secret_id_accessor": []byte(secretIDAccessor),
	}

	roleID, err := e.service.ReadAppRoleRoleID(ctx, params.Backend, params.RoleName)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errReadRoleID)
	}
	cr.Status.AtProvider.RoleID = roleID
	connectionDetails["role_id"] = []byte(roleID)

	return managed.ExternalCreation{ConnectionDetails: connectionDetails}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.AppRoleSecretID)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAppRoleSecretID)
	}

	params := cr.Spec.ForProvider

	oldAccessor := meta.GetExternalName(cr)
	if oldAccessor != "" && oldAccessor != cr.GetName() {
		if err := e.service.DestroyAppRoleSecretIDByAccessor(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName, oldAccessor); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errDestroySecretID)
		}
	} else if oldSecretID, _ := e.getSecretID(ctx, cr); oldSecretID != "" {
		if err := e.service.DestroyAppRoleSecretID(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName, oldSecretID); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errDestroySecretID)
		}
	}

	vaultParams := map[string]interface{}{}
	if len(params.Metadata) > 0 {
		metaJSON, err := json.Marshal(params.Metadata)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "cannot marshal metadata")
		}
		vaultParams["metadata"] = string(metaJSON)
	}
	if len(params.CidrList) > 0 {
		vaultParams["cidr_list"] = params.CidrList
	}
	if len(params.TokenBoundCIDRs) > 0 {
		vaultParams["token_bound_cidrs"] = params.TokenBoundCIDRs
	}

	data, err := e.service.GenerateAppRoleSecretID(ctx, params.Backend, params.RoleName, vaultParams)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateSecretID)
	}

	secretID, _ := data["secret_id"].(string)
	secretIDAccessor, _ := data["secret_id_accessor"].(string)

	meta.SetExternalName(cr, secretIDAccessor)

	connectionDetails := managed.ConnectionDetails{
		"secret_id":          []byte(secretID),
		"secret_id_accessor": []byte(secretIDAccessor),
	}

	roleID, err := e.service.ReadAppRoleRoleID(ctx, params.Backend, params.RoleName)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errReadRoleID)
	}
	cr.Status.AtProvider.RoleID = roleID
	connectionDetails["role_id"] = []byte(roleID)

	return managed.ExternalUpdate{ConnectionDetails: connectionDetails}, nil
}

func (e *external) getSecretID(ctx context.Context, cr *v1beta1.AppRoleSecretID) (string, string) {
	accessor := meta.GetExternalName(cr)
	if accessor != "" && accessor != cr.GetName() {
		data, err := e.service.LookupAppRoleSecretID(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName, accessor)
		if err == nil {
			sid, _ := data["secret_id"].(string)
			return sid, accessor
		}
	}

	if cr.Status.AtProvider.SecretID != "" {
		return cr.Status.AtProvider.SecretID, cr.Status.AtProvider.SecretIDAccessor
	}

	ref := cr.GetWriteConnectionSecretToReference()
	if ref != nil {
		s := &corev1.Secret{}
		nn := types.NamespacedName{Name: ref.Name, Namespace: cr.GetNamespace()}
		if err := e.kube.Get(ctx, nn, s); err == nil {
			return string(s.Data["secret_id"]), string(s.Data["secret_id_accessor"])
		}
	}

	return "", ""
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.AppRoleSecretID)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAppRoleSecretID)
	}

	accessor := meta.GetExternalName(cr)
	if accessor != "" && accessor != cr.GetName() {
		if err := e.service.DestroyAppRoleSecretIDByAccessor(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName, accessor); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDestroySecretID)
		}
		return managed.ExternalDelete{}, nil
	}

	secretID, _ := e.getSecretID(ctx, cr)
	if secretID != "" {
		if err := e.service.DestroyAppRoleSecretID(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.RoleName, secretID); err != nil {
			return managed.ExternalDelete{}, errors.Wrap(err, errDestroySecretID)
		}
	}

	return managed.ExternalDelete{}, nil
}
