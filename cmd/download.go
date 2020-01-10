package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Config struct {
	AutoConfirm bool
	Dir         string
	User        string
	Password    string
	RetryTimes  int
	Host        string
	Key         string
}

var cfg Config

func downloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download docker image.",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	cmd.Flags().SortFlags = false
	addDownloadFlags(cmd.Flags(), cfg)
	return cmd
}

func addDownloadFlags(flagSet *pflag.FlagSet, cfg Config) {
	flagSet.BoolVarP(&cfg.AutoConfirm, "yes", "y", false, "Answer yes for any confirmations.")
	flagSet.StringVarP(&cfg.Dir, "dir", "d", "/var/opt/kubernetes/offline", "Images tar directory path.")
	flagSet.StringVarP(&cfg.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&cfg.Password, "pass", "p", "", "Registry account password.")
	flagSet.StringVar(&cfg.Host, "host", "", "The host name of the registry.")
	flagSet.StringVar(&cfg.Key, "key", "", "Key file registry account.")
	flagSet.IntVarP(&cfg.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
}
