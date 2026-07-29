package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	authmethodv1beta1 "github.com/rossigee/provider-vault/apis/authmethod/v1beta1"
	kvsecretv1beta1 "github.com/rossigee/provider-vault/apis/kvsecret/v1beta1"
	policyv1beta1 "github.com/rossigee/provider-vault/apis/policy/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
)

func AddToScheme(s *runtime.Scheme) error {
	if err := vaultv1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := kvsecretv1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := policyv1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := authmethodv1beta1.AddToScheme(s); err != nil {
		return err
	}
	return nil
}
