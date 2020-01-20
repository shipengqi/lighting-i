package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/filelock"
	"github.com/shipengqi/lighting-i/pkg/log"
)

func addUploadFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&Conf.Org, "organization", "o", "", "Organization name of the images.")
}

func uploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:	_defaultUploadCommand,
		Short:	"Upload docker images.",
		Run: func(cmd *cobra.Command, args []string) {
			defer func() {
				log.Infof("You can refer to %s for more detail.", LogFilePath)
				filelock.UnLock(_defaultDownloadLockFile)
			}()
		},
	}
	cmd.Flags().SortFlags = false
	addUploadFlags(cmd.Flags())
	return cmd
}