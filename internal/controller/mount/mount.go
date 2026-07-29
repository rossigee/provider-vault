package mount

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/mount/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotMount    = "managed resource is not a Mount custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errCreateMount  = "cannot create Vault mount"
	errDeleteMount  = "cannot delete Vault mount"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.MountKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.MountGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.TrackerFn(func(ctx context.Context, mg resource.Managed) error { return nil }),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(recorder.NewNopRecorder()),
		managed.WithDeterministicExternalName(true))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1beta1.Mount{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return nil, errors.New(errNotMount)
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

	svc, err := config.NewClient()
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
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotMount)
	}

	_, err := e.service.GetMount(ctx, cr.Spec.ForProvider.Path)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.Path = cr.Spec.ForProvider.Path
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotMount)
	}

	if err := e.service.EnableMount(ctx,
		cr.Spec.ForProvider.Path,
		cr.Spec.ForProvider.Type,
		cr.Spec.ForProvider.Description,
		cr.Spec.ForProvider.DefaultLeaseTTL,
		cr.Spec.ForProvider.MaxLeaseTTL,
		cr.Spec.ForProvider.Options,
		cr.Spec.ForProvider.Config,
	); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateMount)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotMount)
	}

	if err := e.service.EnableMount(ctx,
		cr.Spec.ForProvider.Path,
		cr.Spec.ForProvider.Type,
		cr.Spec.ForProvider.Description,
		cr.Spec.ForProvider.DefaultLeaseTTL,
		cr.Spec.ForProvider.MaxLeaseTTL,
		cr.Spec.ForProvider.Options,
		cr.Spec.ForProvider.Config,
	); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateMount)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Mount)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotMount)
	}

	if err := e.service.DisableMount(ctx, cr.Spec.ForProvider.Path); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteMount)
	}

	return managed.ExternalDelete{}, nil
}
