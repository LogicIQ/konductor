package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	syncv1 "github.com/LogicIQ/konductor/api/v1"
)

var (
	// Build-time variables
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"

	kubeconfig string
	namespace  string
	logLevel   string
	k8sClient  client.Client
	logger     *zap.Logger
)

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
	rootCmd := &cobra.Command{
		Use:   "koncli",
		Short: "Konductor CLI for coordination primitives",
		Long:  "A CLI tool to interact with Konductor synchronization primitives (semaphores, barriers, leases, gates)\n\nNamespace Detection:\n  - Auto-detects namespace when running in a pod\n  - Falls back to kubeconfig context or 'default'",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initLogger(); err != nil {
				return err
			}
			return initKubeClient(cmd)
		},
	}

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace (auto-detected if running in pod)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	// Bind flags to viper
	viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Set up viper
	viper.SetConfigName("koncli")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.konductor")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("KONCLI")
	viper.AutomaticEnv()

	// Read config file if it exists
	viper.ReadInConfig()

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newOperatorCmd())
	rootCmd.AddCommand(newSemaphoreCmd())
	rootCmd.AddCommand(newBarrierCmd())
	rootCmd.AddCommand(newLeaseCmd())
	rootCmd.AddCommand(newGateCmd())
	rootCmd.AddCommand(newStatusCmd())

	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command execution failed", zap.Error(err))
		return err
	}

	logger.Sync()
	return nil
}

func initLogger() error {
	var level zapcore.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var err error
	logger, err = config.Build()
	return err
}

func initKubeClient(cmd *cobra.Command) error {
	// Get values from viper (which includes flags, config file, and env vars)
	kubeconfig = viper.GetString("kubeconfig")
	namespace = viper.GetString("namespace")
	logLevel = viper.GetString("log-level")

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	scheme, err := syncv1.SchemeBuilder.Build()
	if err != nil {
		return err
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return err
	}

	// Only auto-detect if namespace wasn't explicitly set via flag
	if !cmd.PersistentFlags().Changed("namespace") {
		namespace = detectNamespace()
	}

	return nil
}

func detectNamespace() string {
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		ns := strings.TrimSpace(string(data))
		if ns != "" {
			logger.Debug("Auto-detected namespace from pod service account", zap.String("namespace", ns))
			return ns
		}
	}

	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		logger.Debug("Auto-detected namespace from POD_NAMESPACE env", zap.String("namespace", ns))
		return ns
	}

	if ns := os.Getenv("NAMESPACE"); ns != "" {
		logger.Debug("Auto-detected namespace from NAMESPACE env", zap.String("namespace", ns))
		return ns
	}

	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}

	if config, err := clientcmd.LoadFromFile(kubeconfig); err == nil {
		if config.Contexts[config.CurrentContext] != nil {
			if ns := config.Contexts[config.CurrentContext].Namespace; ns != "" {
				logger.Debug("Auto-detected namespace from kubeconfig context", zap.String("namespace", ns))
				return ns
			}
		}
	}

	logger.Debug("Using default namespace (no auto-detection available)")
	return "default"
}
