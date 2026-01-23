package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initTestLogger() *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	testLogger, err := config.Build()
	if err != nil {
		panic("failed to build test logger: " + err.Error())
	}
	return testLogger
}

func executeCommandWithOutputAndLogs(t *testing.T, cmd *cobra.Command) (string, error) {
	t.Helper()
	// Use local outputFormat to avoid global state mutation
	localOutputFormat := outputFormat
	if localOutputFormat == "" {
		localOutputFormat = "text"
	}

	// Create a buffer to capture logs
	var logBuf bytes.Buffer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = ""

	var encoder zapcore.Encoder
	if localOutputFormat == "json" {
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(&logBuf),
		zapcore.DebugLevel,
	)
	// Store original logger and restore after test
	originalLogger := logger
	logger = zap.New(core)
	defer func() { logger = originalLogger }()

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	return buf.String() + logBuf.String(), err
}

func executeCommandWithOutput(t *testing.T, cmd *cobra.Command) (string, error) {
	return executeCommandWithOutputAndLogs(t, cmd)
}
