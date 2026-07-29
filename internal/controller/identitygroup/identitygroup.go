package identitygroup

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/identitygroup/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotIdentityGroup    = "managed resource is not an IdentityGroup custom resource"
	errTrackPCUsage        = "cannot track ProviderConfig usage"
	errGetPC               = "cannot get ProviderConfig"
	errGetCreds            = "cannot get credentials"
	errCreateIdentityGroup = "cannot create Vault identity group"
	errUpdateIdentityGroup = "cannot update Vault identity group"
	errDeleteIdentityGroup = "cannot delete Vault identity group"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.IdentityGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.IdentityGroupGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.IdentityGroup{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.IdentityGroup)
	if !ok {
		return nil, errors.New(errNotIdentityGroup)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &vaultv1beta1.ProviderConfig{}
	pcRef := cr.GetProviderConfigReference()

	pcName := "default"
	if pcRef != nil && pcRef.Name != "" {
		pcName = pcRef.Name
	}

	pcErr := c.kube.Get(ctx, types.NamespacedName{Name: pcName, Namespace: "crossplane-system"}, pc)
	if pcErr != nil {
		pcNamespace := cr.GetNamespace()
		if pcNamespace != "crossplane-system" {
			fallbackErr := c.kube.Get(ctx, types.NamespacedName{Name: pcName, Namespace: pcNamespace}, pc)
			if fallbackErr != nil {
				return nil, errors.Wrapf(pcErr, "cannot get ProviderConfig '%s'", pcName)
			}
		} else {
			return nil, errors.Wrapf(pcErr, "cannot get ProviderConfig '%s'", pcName)
		}
	}

	config, err := clients.GetConfig(ctx, c.kube, pc)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := clients.NewVaultClientFromConfig(config.Address, config.Token, config.Insecure)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	return &external{service: svc}, nil
}

type external struct {
	service *clients.VaultClient
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.IdentityGroup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIdentityGroup)
	}

	data, err := e.service.GetIdentityGroup(ctx, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name
	if id, ok := data["id"].(string); ok {
		cr.Status.AtProvider.ID = id
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.IdentityGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIdentityGroup)
	}

	p := cr.Spec.ForProvider
	params := map[string]interface{}{
		"name": p.Name,
	}
	if p.Type != "" {
		params["type"] = p.Type
	}
	if len(p.Policies) > 0 {
		params["policies"] = p.Policies
	}
	if len(p.MemberEntityIDs) > 0 {
		params["member_entity_ids"] = p.MemberEntityIDs
	}
	if len(p.Metadata) > 0 {
		params["metadata"] = p.Metadata
	}

	data, err := e.service.CreateIdentityGroup(ctx, params)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateIdentityGroup)
	}

	if id, ok := data["id"].(string); ok {
		cr.Status.AtProvider.ID = id
	}
	cr.Status.AtProvider.Name = p.Name

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.IdentityGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIdentityGroup)
	}

	p := cr.Spec.ForProvider
	params := map[string]interface{}{
		"name": p.Name,
	}
	if len(p.Policies) > 0 {
		params["policies"] = p.Policies
	}
	if len(p.MemberEntityIDs) > 0 {
		params["member_entity_ids"] = p.MemberEntityIDs
	}
	if len(p.Metadata) > 0 {
		params["metadata"] = p.Metadata
	}

	if err := e.service.UpdateIdentityGroup(ctx, p.Name, params); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateIdentityGroup)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.IdentityGroup)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotIdentityGroup)
	}

	if err := e.service.DeleteIdentityGroup(ctx, cr.Spec.ForProvider.Name); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteIdentityGroup)
	}

	return managed.ExternalDelete{}, nil
}
