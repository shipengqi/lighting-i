package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func uploadCommand() *cobra.Command {
	var conf = &DownloadConfig{}
	cmd := &cobra.Command{
		Use:	"upload",
		Short:	"Upload docker images.",
		PreRun: func(cmd *cobra.Command, args []string) {

		},
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	cmd.Flags().SortFlags = false
	addUploadFlags(cmd.Flags(), conf)
	return cmd
}

func addUploadFlags(flagSet *pflag.FlagSet, cfg *DownloadConfig) {
	flagSet.BoolVarP(&cfg.AutoConfirm, "yes", "y", false, "Answer yes for any confirmations.")
	flagSet.StringVarP(&cfg.Dir, "dir", "d", "/var/opt/kubernetes/offline", "Images tar directory path.")
	flagSet.StringVarP(&cfg.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&cfg.Password, "pass", "p", "", "Registry account password.")
	flagSet.StringVar(&cfg.Registry, "host", "", "The host name of the registry.")
	flagSet.StringVar(&cfg.Key, "key", "", "Key file registry account.")
	flagSet.IntVarP(&cfg.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
}