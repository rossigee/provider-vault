package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/alecthomas/kingpin/v2"
	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	approlesecretidv1beta1 "github.com/rossigee/provider-vault/apis/approlesecretid/v1beta1"
	auditdevicev1beta1 "github.com/rossigee/provider-vault/apis/auditdevice/v1beta1"
	authbackendrolev1beta1 "github.com/rossigee/provider-vault/apis/authbackendrole/v1beta1"
	authmethodv1beta1 "github.com/rossigee/provider-vault/apis/authmethod/v1beta1"
	awsauthconfigv1beta1 "github.com/rossigee/provider-vault/apis/awsauthconfig/v1beta1"
	azureauthconfigv1beta1 "github.com/rossigee/provider-vault/apis/azureauthconfig/v1beta1"
	certificatev1beta1 "github.com/rossigee/provider-vault/apis/certificate/v1beta1"
	databasebackendv1beta1 "github.com/rossigee/provider-vault/apis/databasebackend/v1beta1"
	databaserolev1beta1 "github.com/rossigee/provider-vault/apis/databaserole/v1beta1"
	gcpauthconfigv1beta1 "github.com/rossigee/provider-vault/apis/gcpauthconfig/v1beta1"
	identityentityv1beta1 "github.com/rossigee/provider-vault/apis/identityentity/v1beta1"
	identitygroupv1beta1 "github.com/rossigee/provider-vault/apis/identitygroup/v1beta1"
	jwtauthconfigv1beta1 "github.com/rossigee/provider-vault/apis/jwtauthconfig/v1beta1"
	kubernetesauthconfigv1beta1 "github.com/rossigee/provider-vault/apis/kubernetesauthconfig/v1beta1"
	kvsecretv1beta1 "github.com/rossigee/provider-vault/apis/kvsecret/v1beta1"
	ldapauthconfigv1beta1 "github.com/rossigee/provider-vault/apis/ldapauthconfig/v1beta1"
	leaserenewalv1beta1 "github.com/rossigee/provider-vault/apis/leaserenewal/v1beta1"
	mountv1beta1 "github.com/rossigee/provider-vault/apis/mount/v1beta1"
	namespacesv1beta1 "github.com/rossigee/provider-vault/apis/namespaces/v1beta1"
	pkiconfigv1beta1 "github.com/rossigee/provider-vault/apis/pkiconfig/v1beta1"
	policyv1beta1 "github.com/rossigee/provider-vault/apis/policy/v1beta1"
	quotav1beta1 "github.com/rossigee/provider-vault/apis/quota/v1beta1"
	secretbackendrolev1beta1 "github.com/rossigee/provider-vault/apis/secretbackendrole/v1beta1"
	tokenv1beta1 "github.com/rossigee/provider-vault/apis/token/v1beta1"
	transitkeyv1beta1 "github.com/rossigee/provider-vault/apis/transitkey/v1beta1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/rossigee/provider-vault/apis"
	"github.com/rossigee/provider-vault/internal/controller"
	"github.com/rossigee/provider-vault/internal/version"
)

func main() {
	var (
		app                      = kingpin.New(filepath.Base(os.Args[0]), "Vault Crossplane provider").DefaultEnvars()
		debug                    = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncInterval             = app.Flag("sync", "Sync interval controls how often all resources will be double checked for drift.").Short('s').Default("1h").Duration()
		pollInterval             = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("1m").Duration()
		leaderElection           = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		maxReconcileRate         = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("100").Int()
		pollStateMetricInterval  = app.Flag("poll-state-metric", "State metric recording interval").Default("5s").Duration()
		metricsBindAddress       = app.Flag("metrics-bind-address", "The address the metrics endpoint binds to.").Default(":8080").String()
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

	cacheOpts := cache.Options{}
	if namespace != "" {
		cacheOpts.DefaultNamespaces = map[string]cache.Config{namespace: {}}
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:                *leaderElection,
		LeaderElectionID:              "crossplane-leader-election-provider-vault",
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		Metrics: metricserver.Options{
			BindAddress: *metricsBindAddress,
		},
		Cache: cacheOpts,
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add Vault APIs to scheme")

	rateLimiter := ratelimiter.NewGlobal(*maxReconcileRate)

	mrStateMetrics := statemetrics.NewMRStateMetrics()
	metrics.Registry.MustRegister(mrStateMetrics)

	mo := xpcontroller.MetricOptions{
		PollStateMetricInterval: *pollStateMetricInterval,
		MRStateMetrics:          mrStateMetrics,
	}

	o := xpcontroller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       rateLimiter,
		MetricOptions:           &mo,
	}

	kingpin.FatalIfError(controller.Setup(mgr, o), "Cannot setup Vault controllers")

	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &approlesecretidv1beta1.AppRoleSecretIDList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for AppRoleSecretID")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &auditdevicev1beta1.AuditDeviceList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for AuditDevice")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &authbackendrolev1beta1.AuthBackendRoleList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for AuthBackendRole")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &authmethodv1beta1.AuthMethodList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for AuthMethod")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &awsauthconfigv1beta1.AWSAuthConfigList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for AWSAuthConfig")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &azureauthconfigv1beta1.AzureAuthConfigList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for AzureAuthConfig")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &certificatev1beta1.CertificateList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Certificate")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &databasebackendv1beta1.DatabaseBackendList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for DatabaseBackend")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &databaserolev1beta1.DatabaseRoleList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for DatabaseRole")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &gcpauthconfigv1beta1.GCPAuthConfigList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for GCPAuthConfig")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &identityentityv1beta1.IdentityEntityList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for IdentityEntity")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &identitygroupv1beta1.IdentityGroupList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for IdentityGroup")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &jwtauthconfigv1beta1.JWTAuthConfigList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for JWTAuthConfig")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &kubernetesauthconfigv1beta1.KubernetesAuthConfigList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for KubernetesAuthConfig")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &kvsecretv1beta1.KVSecretList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for KVSecret")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &ldapauthconfigv1beta1.LDAPAuthConfigList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for LDAPAuthConfig")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &leaserenewalv1beta1.LeaseRenewalList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for LeaseRenewal")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &mountv1beta1.MountList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Mount")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &namespacesv1beta1.VaultNamespaceList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for VaultNamespace")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &pkiconfigv1beta1.PKIConfigList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for PKIConfig")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &policyv1beta1.PolicyList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Policy")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &quotav1beta1.QuotaList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Quota")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &secretbackendrolev1beta1.SecretBackendRoleList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for SecretBackendRole")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &tokenv1beta1.TokenList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for Token")
	kingpin.FatalIfError(mgr.Add(statemetrics.NewMRStateRecorder(mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &transitkeyv1beta1.TransitKeyList{}, o.MetricOptions.PollStateMetricInterval)), "Cannot register state metrics for TransitKey")

	log.Info("Starting manager")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}

func getWatchNamespace() (string, error) {
	ns, found := os.LookupEnv("WATCH_NAMESPACE")
	if found && ns != "" {
		return ns, nil
	}
	return "", nil
}
