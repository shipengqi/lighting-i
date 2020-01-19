package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/filelock"
)

func addUploadFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&Conf.Org, "organization", "o", "", "Organization name of the images.")
}

func uploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:	"upload",
		Short:	"Upload docker images.",
		Run: func(cmd *cobra.Command, args []string) {
			defer filelock.UnLock(_defaultUploadLockFile)
		},
	}
	cmd.Flags().SortFlags = false
	addUploadFlags(cmd.Flags())
	return cmd
}