package clients

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
)

const errGetCreds = "cannot get credentials"

// resourceRef is the subset of the managed resource interface needed to resolve
// the referenced ProviderConfig and record its usage.
type resourceRef interface {
	client.Object
	GetProviderConfigReference() *xpv1.ProviderConfigReference
}

// TrackUsage records that the managed resource is using the ProviderConfig it
// references by creating or updating a ProviderConfigUsage. It mirrors
// resource.ProviderConfigUsageTracker using the typed ProviderConfigReference
// accessors generated for this provider's resources.
func TrackUsage(ctx context.Context, kube client.Client, mg resourceRef) error {
	ref := mg.GetProviderConfigReference()
	if ref == nil || ref.Name == "" {
		return nil
	}

	gvk := mg.GetObjectKind().GroupVersionKind()
	pcu := &vaultv1beta1.ProviderConfigUsage{}
	pcu.SetName(string(mg.GetUID()))
	pcu.SetNamespace(mg.GetNamespace())
	pcu.SetLabels(map[string]string{
		xpv1.LabelKeyProviderName: ref.Name,
		xpv1.LabelKeyProviderKind: ref.Kind,
	})
	pcu.SetOwnerReferences([]metav1.OwnerReference{meta.AsController(meta.TypedReferenceTo(mg, gvk))})
	pcu.SetProviderConfigReference(xpv1.Reference{Name: ref.Name})
	pcu.SetResourceReference(xpv1.TypedReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       mg.GetName(),
	})

	return errors.Wrap(resource.Ignore(resource.IsNotAllowed, resource.NewAPIUpdatingApplicator(kube).Apply(ctx, pcu,
		resource.MustBeControllableBy(mg.GetUID()),
		resource.AllowUpdateIf(func(current, _ runtime.Object) bool {
			return current.(*vaultv1beta1.ProviderConfigUsage).GetProviderConfigReference() != pcu.GetProviderConfigReference()
		}),
	)), "cannot apply ProviderConfigUsage")
}

// Connect builds a Vault client from the ProviderConfig referenced by the
// supplied managed resource. A ProviderConfig named "default" is used when the
// resource does not reference one, mirroring Crossplane conventions. The
// ProviderConfig is looked up in the crossplane-system namespace first, then in
// the namespace of the managed resource itself.
func Connect(ctx context.Context, kube client.Client, mg resourceRef) (*VaultClient, error) {
	pcName := "default"
	if ref := mg.GetProviderConfigReference(); ref != nil && ref.Name != "" {
		pcName = ref.Name
	}

	pc := &vaultv1beta1.ProviderConfig{}
	err := kube.Get(ctx, client.ObjectKey{Name: pcName, Namespace: "crossplane-system"}, pc)
	if err != nil {
		if ns := mg.GetNamespace(); ns != "crossplane-system" {
			if err2 := kube.Get(ctx, client.ObjectKey{Name: pcName, Namespace: ns}, pc); err2 == nil {
				err = nil
			}
		}
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get ProviderConfig '%s'", pcName)
		}
	}

	config, err := GetConfig(ctx, kube, pc)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := config.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	return svc, nil
}
