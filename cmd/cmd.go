package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	_defaultBaseDir          = "/var/opt/lighting"
	_defaultImageSet         = _defaultBaseDir + "/image_set.yaml"
	_defaultImagesDir        = _defaultBaseDir + "/offline"
	_defaultDownloadLockFile = _defaultBaseDir + "/images.download.lock"
	_defaultUploadLockFile   = _defaultBaseDir + "/images.upload.lock"
	_defaultManifestJson     = "manifest.json"
)

func NewLightingCommand() *cobra.Command {
	lightingCmd := &cobra.Command{
		Use:   "lighting",
		Short: "lighting is used to bulk download or upload docker images. It's much faster than 'docker pull'",
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


func handleSignals(exitCh chan int) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	for {
		 s := <-c
		switch s {
		case syscall.SIGINT: // kill -SIGINT XXXX or Ctrl+c
			fmt.Println("[SIGNAL] Catch SIGINT")
			exitCh <- 0

		case syscall.SIGTERM: // kill -SIGTERM XXXX
			fmt.Println("[SIGNAL] Catch SIGTERM")
			exitCh <- 1

		case syscall.SIGQUIT: // kill -SIGQUIT XXXX
			fmt.Println("[SIGNAL] Catch SIGQUIT")
			exitCh <- 0
		default:
			return
		}
	}
}