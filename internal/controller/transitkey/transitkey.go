package transitkey

import (
	"context"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	v1beta1 "github.com/rossigee/provider-vault/apis/transitkey/v1beta1"
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
	errRotateTransitKey = "cannot rotate Vault transit key"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.TransitKeyKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.TransitKeyGroupVersionKind),
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
		For(&v1beta1.TransitKey{}).
		Complete(r)
}

type connector struct {
	kube client.Client
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1beta1.TransitKey)
	if !ok {
		return nil, errors.New(errNotTransitKey)
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
	cr, ok := mg.(*v1beta1.TransitKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTransitKey)
	}

	data, err := e.service.GetTransitKey(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
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

	p := cr.Spec.ForProvider
	upToDate := true
	if p.Type != "" {
		if t, ok := data["type"].(string); ok && t != p.Type {
			upToDate = false
		}
	}
	if p.ConvergentEncryption != nil {
		if b, ok := data["convergent_encryption"].(bool); ok && b != *p.ConvergentEncryption {
			upToDate = false
		}
	}
	if p.Derived != nil {
		if b, ok := data["derived"].(bool); ok && b != *p.Derived {
			upToDate = false
		}
	}
	if p.Exportable != nil {
		if b, ok := data["exportable"].(bool); ok && b != *p.Exportable {
			upToDate = false
		}
	}
	if p.AllowPlaintextBackup != nil {
		if b, ok := data["allow_plaintext_backup"].(bool); ok && b != *p.AllowPlaintextBackup {
			upToDate = false
		}
	}
	if p.MinDecryptionVersion != nil {
		if f, ok := data["min_decryption_version"].(float64); ok && int(f) != *p.MinDecryptionVersion {
			upToDate = false
		}
	}
	if p.MinEncryptionVersion != nil {
		if f, ok := data["min_encryption_version"].(float64); ok && int(f) != *p.MinEncryptionVersion {
			upToDate = false
		}
	}
	if p.AutoRotatePeriod != "" {
		if d, err := time.ParseDuration(p.AutoRotatePeriod); err == nil {
			if f, ok := data["auto_rotate_period"].(float64); ok && int64(f) != int64(d.Seconds()) {
				upToDate = false
			}
		}
	}
	if p.RotateToVersion != nil && cr.Status.AtProvider.LatestVersion < *p.RotateToVersion {
		upToDate = false
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
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

	// The key must be marked as allowed for deletion so that Delete can remove
	// it when the managed resource is deleted.
	params["deletion_allowed"] = true

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
	if cr.Spec.ForProvider.MinDecryptionVersion != nil {
		params["min_decryption_version"] = *cr.Spec.ForProvider.MinDecryptionVersion
	}
	if cr.Spec.ForProvider.MinEncryptionVersion != nil {
		params["min_encryption_version"] = *cr.Spec.ForProvider.MinEncryptionVersion
	}
	if cr.Spec.ForProvider.AutoRotatePeriod != "" {
		params["auto_rotate_period"] = cr.Spec.ForProvider.AutoRotatePeriod
	}
	if len(params) > 0 {
		if err := e.service.ConfigureTransitKey(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name, params); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errCreateTransitKey)
		}
	}

	// Rotate until the key's latest version reaches the desired target. Each
	// rotation increments the latest version by one. The latest version was
	// recorded in Observe, so the number of rotations needed is the difference
	// between the target and the observed version.
	if target := cr.Spec.ForProvider.RotateToVersion; target != nil {
		rotations := *target - cr.Status.AtProvider.LatestVersion
		if rotations > 0 {
			for i := 0; i < rotations; i++ {
				if err := e.service.RotateTransitKey(ctx, cr.Spec.ForProvider.Backend, cr.Spec.ForProvider.Name); err != nil {
					return managed.ExternalUpdate{}, errors.Wrap(err, errRotateTransitKey)
				}
			}
		}
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
