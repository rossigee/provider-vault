//go:build generate
// +build generate

package apis

import (
	_ "github.com/crossplane/crossplane-tools/cmd/angryjet"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)

//go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../hack/boilerplate.go.txt paths=./... crd:allowDangerousTypes=true output:artifacts:config=../package/crds

//go:generate go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet generate-methodsets --header-file=../hack/boilerplate.go.txt ./...
