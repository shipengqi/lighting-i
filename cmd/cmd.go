package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	_defaultBaseDir = "/var/opt/lighting/"
	_defaultImagesDir = _defaultBaseDir + "offline"
	_defaultDownloadLockFile = _defaultBaseDir + ".download.lock"
	_defaultUploadLockFile = _defaultBaseDir + ".upload.lock"
)

func NewLightingCommand() *cobra.Command {
	lightingCmd := &cobra.Command{
		Use:	"lighting",
		Short:	"suite-installer internal API server.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
				_ = cmd.Help()
				os.Exit(0)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Disable commands sorting
	cobra.EnableCommandSorting = false
	// Reset Flags
	lightingCmd.ResetFlags()
	// Add sub commands
	lightingCmd.AddCommand(downloadCommand())
	lightingCmd.AddCommand(uploadCommand())
	return lightingCmd
}
