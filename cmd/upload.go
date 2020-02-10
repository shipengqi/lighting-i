package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/filelock"
	"github.com/shipengqi/lighting-i/pkg/log"
)

func addUploadFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&Conf.Registry, "registry", "r", "https://registry-1.docker.io", "The host of the registry.")
	flagSet.StringVarP(&Conf.Org, "organization", "o", "", "Organization name of the images.")
	flagSet.StringVarP(&Conf.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&Conf.Password, "pass", "p", "", "Registry account password.")
	flagSet.IntVarP(&Conf.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
	flagSet.StringVarP(&Conf.Dir, "dir", "d", _defaultImagesDir, "Images tar directory path.")
	flagSet.BoolVarP(&Conf.AutoConfirm, "yes", "y", false, "Answer yes for any confirmations.")
	flagSet.BoolVarP(&Conf.Force, "force", "f", false, "If true, ignore the process lock.")
}

func uploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:	_defaultUploadCommand,
		Aliases: []string{_defaultUploadAlias},
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