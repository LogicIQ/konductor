package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func executeCommandWithOutput(t *testing.T, cmd *cobra.Command) (string, error) {
	// Create a buffer to capture logs
	var logBuf bytes.Buffer
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = ""
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
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
