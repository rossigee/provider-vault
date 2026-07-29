package main

import (
	"os"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/rossigee/provider-vault/apis"
	"github.com/rossigee/provider-vault/internal/controller"
	"github.com/rossigee/provider-vault/internal/version"
)

func main() {
	logging.SetLogrusFormatter(logging.NewLogrusFormatter("provider-vault", version.Version))

	_ = zap.Options{
		Development: true,
	}

	opts := ctrl.Options{
		Scheme:           apis.Scheme,
		SyncPeriod:       nil,
		LeaderElection:   false,
		LeaderElectionID: "crossplane.io/provider-vault",
		CertDir:          os.Getenv("WEBHOOK_TLS_CERT_DIR"),
		WebhookServer: webhook.NewServer(webhook.Options{
			CertDir: os.Getenv("WEBHOOK_TLS_CERT_DIR"),
		}),
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		panic(err)
	}

	if err := xpv1.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	o := controller.Options{
		Logger:           logging.NewLogger("provider-vault"),
		PollInterval:     nil,
		MaxReconcileRate: 0,
	}

	if err := controller.Setup(mgr, o); err != nil {
		panic(err)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		panic(err)
	}
}
