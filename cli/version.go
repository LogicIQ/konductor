package main

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("Version info",
				zap.String("version", version),
				zap.String("commit", commit),
				zap.String("built", buildDate),
			)
		},
	}

	return cmd
}
