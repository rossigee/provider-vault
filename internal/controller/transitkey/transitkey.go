package transitkey

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1beta1 "github.com/rossigee/provider-vault/apis/transitkey/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
	"github.com/rossigee/provider-vault/internal/clients"
	"github.com/rossigee/provider-vault/internal/recorder"
)

const (
	errNotTransitKey    = "managed resource is not a TransitKey custom resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errCreateTransitKey = "cannot create Vault transit key"
	errDeleteTransitKey = "cannot delete Vault transit key"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.TransitKeyKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.TransitKeyGroupVersionKind),
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
		For(&v1beta1.TransitKey{}).
		Complete(r)
}

type connector struct {
	kube  client.Client
	usage resource.TrackerFn
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.TransitKey)
	if !ok {
		return nil, errors.New(errNotTransitKey)
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
	cr, ok := mg.(*v1beta1.TransitKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTransitKey)
	}

	data, err := e.service.GetTransitKey(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.Name = cr.Spec.ForProvider.Name
	if t, ok := data["type"].(string); ok {
		cr.Status.AtProvider.Type = t
	}
	if da, ok := data["deletion_allowed"].(bool); ok {
		cr.Status.AtProvider.DeletionAllowed = da
	}
	if mdv, ok := data["min_decryption_version"].(float64); ok {
		cr.Status.AtProvider.MinDecryptionVersion = int(mdv)
	}
	if lv, ok := data["latest_version"].(float64); ok {
		cr.Status.AtProvider.LatestVersion = int(lv)
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.TransitKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTransitKey)
	}

	params := make(map[string]interface{})
	if cr.Spec.ForProvider.Type != "" {
		params["type"] = cr.Spec.ForProvider.Type
	}
	if cr.Spec.ForProvider.ConvergentEncryption != nil {
		params["convergent_encryption"] = *cr.Spec.ForProvider.ConvergentEncryption
	}
	if cr.Spec.ForProvider.Derived != nil {
		params["derived"] = *cr.Spec.ForProvider.Derived
	}
	if cr.Spec.ForProvider.Exportable != nil {
		params["exportable"] = *cr.Spec.ForProvider.Exportable
	}
	if cr.Spec.ForProvider.AllowPlaintextBackup != nil {
		params["allow_plaintext_backup"] = *cr.Spec.ForProvider.AllowPlaintextBackup
	}
	if cr.Spec.ForProvider.AutoRotatePeriod != "" {
		params["auto_rotate_period"] = cr.Spec.ForProvider.AutoRotatePeriod
	}

	if err := e.service.CreateTransitKey(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name, params); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateTransitKey)
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.TransitKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTransitKey)
	}

	params := make(map[string]interface{})
	params["min_decryption_version"] = 0
	params["min_encryption_version"] = 0
	if cr.Spec.ForProvider.MinDecryptionVersion != nil {
		params["min_decryption_version"] = *cr.Spec.ForProvider.MinDecryptionVersion
	}
	if cr.Spec.ForProvider.MinEncryptionVersion != nil {
		params["min_encryption_version"] = *cr.Spec.ForProvider.MinEncryptionVersion
	}
	if cr.Spec.ForProvider.AutoRotatePeriod != "" {
		params["auto_rotate_period"] = cr.Spec.ForProvider.AutoRotatePeriod
	}
	params["deletion_allowed"] = true

	if err := e.service.CreateTransitKey(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name, params); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errCreateTransitKey)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.TransitKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotTransitKey)
	}

	if err := e.service.DeleteTransitKey(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errDeleteTransitKey)
	}

	return managed.ExternalDelete{}, nil
}
