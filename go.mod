module github.com/rossigee/provider-vault

go 1.26.5

require (
	github.com/crossplane/crossplane-runtime/v2 v2.4.0-rc.0
	github.com/crossplane/crossplane v1.20.0-rc.0
	github.com/google/go-cmp v0.6.0
	github.com/pkg/errors v0.9.1
	k8s.io/api v0.36.3
	k8s.io/apimachinery v0.36.3
	k8s.io/client-go v0.36.3
	sigs.k8s.io/controller-runtime v0.24.1
)

replace github.com/crossplane/crossplane-runtime/v2 => github.com/rossigee/crossplane-runtime/v2 v2.4.0-rc.0.0.20260726062756-089a6b3db2f8
