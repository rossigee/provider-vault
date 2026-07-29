package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	authmethodv1beta1 "github.com/rossigee/provider-vault/apis/authmethod/v1beta1"
	kvsecretv1beta1 "github.com/rossigee/provider-vault/apis/kvsecret/v1beta1"
	policyv1beta1 "github.com/rossigee/provider-vault/apis/policy/v1beta1"
	vaultv1beta1 "github.com/rossigee/provider-vault/apis/v1beta1"
)

var Scheme = runtime.NewScheme()

func init() {
	_ = vaultv1beta1.AddToScheme(Scheme)
	_ = kvsecretv1beta1.AddToScheme(Scheme)
	_ = policyv1beta1.AddToScheme(Scheme)
	_ = authmethodv1beta1.AddToScheme(Scheme)
}
