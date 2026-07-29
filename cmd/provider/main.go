package main

import (
	"os"
	"path/filepath"
	"runtime"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/alecthomas/kingpin/v2"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/rossigee/provider-vault/apis"
	"github.com/rossigee/provider-vault/internal/controller"
	"github.com/rossigee/provider-vault/internal/version"
)

func main() {
	var (
		app              = kingpin.New(filepath.Base(os.Args[0]), "Vault Crossplane provider").DefaultEnvars()
		debug            = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncInterval     = app.Flag("sync", "Sync interval controls how often all resources will be double checked for drift.").Short('s').Default("1h").Duration()
		pollInterval     = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("1m").Duration()
		leaderElection   = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("100").Int()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-vault"))

	if *debug {
		ctrl.SetLogger(zl)
	} else {
		ctrl.SetLogger(zl.WithValues("source", "controller-runtime").V(1))
	}

	log.Info("Provider starting up",
		"provider", "provider-vault",
		"version", version.Version,
		"go-version", runtime.Version(),
		"platform", runtime.GOOS+"/"+runtime.GOARCH,
		"sync-interval", syncInterval.String(),
		"poll-interval", pollInterval.String(),
		"max-reconcile-rate", *maxReconcileRate,
		"leader-election", *leaderElection,
		"debug-mode", *debug)

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	namespace, err := getWatchNamespace()
	kingpin.FatalIfError(err, "Cannot get watch namespace")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:                *leaderElection,
		LeaderElectionID:              "crossplane-leader-election-provider-vault",
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		Metrics: server.Options{
			BindAddress: ":8080",
		},
		Cache: cache.Options{DefaultNamespaces: map[string]cache.Config{namespace: {}}},
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add Vault APIs to scheme")

	rateLimiter := ratelimiter.NewGlobal(*maxReconcileRate)

	o := xpcontroller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       rateLimiter,
	}

	kingpin.FatalIfError(controller.Setup(mgr, o), "Cannot setup Vault controllers")

	log.Info("Starting manager")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}

func getWatchNamespace() (string, error) {
	ns, found := os.LookupEnv("WATCH_NAMESPACE")
	if !found {
		return "", nil
	}
	return ns, nil
}
