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

	kubeconfig   string
	namespace    string
	logLevel     string
	outputFormat string
	k8sClient    client.Client
	logger       *zap.Logger
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
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, json)")

	// Bind flags to viper - errors only occur if flag doesn't exist, which can't happen here
	_ = viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

	// Set up viper
	viper.SetConfigName("koncli")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.konductor")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("KONCLI")
	viper.AutomaticEnv()

	// Read config file if it exists (ignore error if file not found)
	_ = viper.ReadInConfig()

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newOperatorCmd())
	rootCmd.AddCommand(newSemaphoreCmd())
	rootCmd.AddCommand(newBarrierCmd())
	rootCmd.AddCommand(newLeaseCmd())
	rootCmd.AddCommand(newGateCmd())
	rootCmd.AddCommand(newMutexCmd())
	rootCmd.AddCommand(newRWMutexCmd())
	rootCmd.AddCommand(newOnceCmd())
	rootCmd.AddCommand(newWaitGroupCmd())
	rootCmd.AddCommand(newStatusCmd())

	if err := rootCmd.Execute(); err != nil {
		if logger != nil {
			logger.Error("Command execution failed", zap.Error(err))
		}
		return err
	}

	if logger != nil {
		if err := logger.Sync(); err != nil {
			// Ignore sync errors on stdout/stderr (common on some platforms)
			// Only log the error, don't fail the command
			logger.Debug("Logger sync failed (non-fatal)", zap.Error(err))
		}
	}
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

	var config zap.Config
	if strings.ToLower(outputFormat) == "json" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.TimeKey = ""
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.Level = zap.NewAtomicLevelAt(level)
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	var err error
	logger, err = config.Build()
	return err
}

func initKubeClient(cmd *cobra.Command) error {
	// Get values from viper (which includes flags, config file, and env vars)
	kubeconfig = viper.GetString("kubeconfig")
	namespace = viper.GetString("namespace")
	logLevel = viper.GetString("log-level")
	outputFormat = viper.GetString("output")

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
	// Try pod service account
	if ns := readPodNamespace(); ns != "" {
		logger.Debug("Auto-detected namespace from pod service account", zap.String("namespace", ns))
		return ns
	}

	// Try environment variables
	if ns := getNamespaceFromEnv(); ns != "" {
		logger.Debug("Auto-detected namespace from environment", zap.String("namespace", ns))
		return ns
	}

	// Try kubeconfig
	if ns := getNamespaceFromKubeconfig(); ns != "" {
		logger.Debug("Auto-detected namespace from kubeconfig context", zap.String("namespace", ns))
		return ns
	}

	logger.Debug("Using default namespace (no auto-detection available)")
	return "default"
}

func readPodNamespace() string {
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

func getNamespaceFromEnv() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return os.Getenv("NAMESPACE")
}

func getNamespaceFromKubeconfig() string {
	kubeconfigPath := kubeconfig
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.RecommendedHomeFile
	}

	if config, err := clientcmd.LoadFromFile(kubeconfigPath); err == nil {
		if config.Contexts[config.CurrentContext] != nil {
			return config.Contexts[config.CurrentContext].Namespace
		}
	}
	return ""
}
