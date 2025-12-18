package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func executeCommandWithOutput(t *testing.T, cmd *cobra.Command) (string, error) {
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()
	w.Close()
	os.Stdout = oldStdout

	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r)
	return buf.String() + stdoutBuf.String(), err
}