package cmd

import (
	"github.com/spf13/cobra"

	"github.com/shipengqi/lighting-i/pkg/filelock"
)

func uploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:	"upload",
		Short:	"Upload docker images.",
		Run: func(cmd *cobra.Command, args []string) {
			defer filelock.UnLock(_defaultUploadLockFile)
		},
	}
	cmd.Flags().SortFlags = false
	return cmd
}