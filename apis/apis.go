package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	authbackendrolev1beta1 "github.com/rossigee/provider-vault/apis/authbackendrole/v1beta1"
	authmethodv1beta1 "github.com/rossigee/provider-vault/apis/authmethod/v1beta1"
	certificatev1beta1 "github.com/rossigee/provider-vault/apis/certificate/v1beta1"
	databaserolev1beta1 "github.com/rossigee/provider-vault/apis/databaserole/v1beta1"
	identityentityv1beta1 "github.com/rossigee/provider-vault/apis/identityentity/v1beta1"
	identitygroupv1beta1 "github.com/rossigee/provider-vault/apis/identitygroup/v1beta1"
	kvsecretv1beta1 "github.com/rossigee/provider-vault/apis/kvsecret/v1beta1"
	mountv1beta1 "github.com/rossigee/provider-vault/apis/mount/v1beta1"
	pkiconfigv1beta1 "github.com/rossigee/provider-vault/apis/pkiconfig/v1beta1"
	policyv1beta1 "github.com/rossigee/provider-vault/apis/policy/v1beta1"
	secretbackendrolev1beta1 "github.com/rossigee/provider-vault/apis/secretbackendrole/v1beta1"
	tokenv1beta1 "github.com/rossigee/provider-vault/apis/token/v1beta1"
	transitkeyv1beta1 "github.com/rossigee/provider-vault/apis/transitkey/v1beta1"
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
	if err := mountv1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := secretbackendrolev1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := authbackendrolev1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := pkiconfigv1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := certificatev1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := databaserolev1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := transitkeyv1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := identityentityv1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := identitygroupv1beta1.AddToScheme(s); err != nil {
		return err
	}
	if err := tokenv1beta1.AddToScheme(s); err != nil {
		return err
	}
	return nil
}
