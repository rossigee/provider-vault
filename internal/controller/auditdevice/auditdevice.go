package auditdevice

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/auditdevice/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotAuditDevice    = "managed resource is not an AuditDevice custom resource"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"
	errGetCreds          = "cannot get credentials"
	errEnableAuditDevice = "cannot enable Vault audit device"
	errDeleteAuditDevice = "cannot disable Vault audit device"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.AuditDeviceKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.AuditDeviceGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.AuditDevice{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.AuditDevice)
	if !ok {
		return nil, errors.New(errNotAuditDevice)
	}

	if err := clients.TrackUsage(ctx, c.kube, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.Connect(ctx, c.kube, cr)
	if err != nil {
		return nil, err
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
	cr, ok := mg.(*v1beta1.AuditDevice)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAuditDevice)
	}

	_, err := e.service.GetAuditDevice(ctx, cr.Spec.ForProvider.Path)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Path = cr.Spec.ForProvider.Path

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.AuditDevice)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAuditDevice)
	}

	p := cr.Spec.ForProvider
	local := false
	if p.Local != nil {
		local = *p.Local
	}

	if err := e.service.EnableAuditDevice(ctx, p.Path, p.Type, p.Description, local, p.Options); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errEnableAuditDevice)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.AuditDevice)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAuditDevice)
	}

	p := cr.Spec.ForProvider
	local := false
	if p.Local != nil {
		local = *p.Local
	}

	if err := e.service.EnableAuditDevice(ctx, p.Path, p.Type, p.Description, local, p.Options); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errEnableAuditDevice)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.AuditDevice)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAuditDevice)
	}

	if err := e.service.DisableAuditDevice(ctx, cr.Spec.ForProvider.Path); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteAuditDevice)
	}

	return managed.ExternalDelete{}, nil
}
