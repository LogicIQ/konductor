package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
	"github.com/LogicIQ/konductor/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	version  = "dev"
)

func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn", "warning":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		return nil, fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", level)
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	return config.Build()
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(syncv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func versionHealthCheck() healthz.Checker {
	return func(req *http.Request) error {
		return nil
	}
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var logLevel string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	// Initialize zap logger
	logger, err := initLogger(logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Set controller-runtime logger
	ctrl.SetLogger(zapr.NewLogger(logger))

	logger.Info("Starting konductor operator",
		zap.String("version", version),
		zap.String("log-level", logLevel),
		zap.Bool("leader-election", enableLeaderElection))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "konductor.io",
		WebhookServer:          nil,
	})
	if err != nil {
		logger.Error("Unable to start manager", zap.Error(err))
		os.Exit(1)
	}

	if err = (&controllers.SemaphoreReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("Unable to create controller", zap.Error(err), zap.String("controller", "Semaphore"))
		os.Exit(1)
	}

	if err = (&controllers.BarrierReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("Unable to create controller", zap.Error(err), zap.String("controller", "Barrier"))
		os.Exit(1)
	}

	if err = (&controllers.LeaseReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("Unable to create controller", zap.Error(err), zap.String("controller", "Lease"))
		os.Exit(1)
	}

	if err = (&controllers.GateReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("Unable to create controller", zap.Error(err), zap.String("controller", "Gate"))
		os.Exit(1)
	}

	if err = (&controllers.MutexReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("Unable to create controller", zap.Error(err), zap.String("controller", "Mutex"))
		os.Exit(1)
	}

	if err = (&controllers.RWMutexReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("Unable to create controller", zap.Error(err), zap.String("controller", "RWMutex"))
		os.Exit(1)
	}

	if err = (&controllers.OnceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("Unable to create controller", zap.Error(err), zap.String("controller", "Once"))
		os.Exit(1)
	}

	if err = (&controllers.WaitGroupReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("Unable to create controller", zap.Error(err), zap.String("controller", "WaitGroup"))
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", versionHealthCheck()); err != nil {
		logger.Error("Unable to set up health check", zap.Error(err))
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", versionHealthCheck()); err != nil {
		logger.Error("Unable to set up ready check", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("Problem running manager", zap.Error(err))
		os.Exit(1)
	}
}
