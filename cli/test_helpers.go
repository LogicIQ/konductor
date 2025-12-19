package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func executeCommandWithOutput(t *testing.T, cmd *cobra.Command) (string, error) {
	// Initialize outputFormat for tests
	if outputFormat == "" {
		outputFormat = "text"
	}

	// Create a buffer to capture logs
	var logBuf bytes.Buffer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = ""

	var encoder zapcore.Encoder
	if outputFormat == "json" {
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
	logger = zap.New(core)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	return buf.String() + logBuf.String(), err
}
