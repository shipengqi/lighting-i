package cmd

import (
	"fmt"
	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"math/rand"
	"sync"
	"time"
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

var steps = []string{
	"downloading source",
	"installing deps",
	"compiling",
	"packaging",
	"seeding database",
	"deploying",
	"staring servers",
	"completed",
}

func downloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download docker image.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("==============")
			uiprogress.Start()

			var wg sync.WaitGroup
			wg.Add(1)
			go deploy("app1", &wg)
			wg.Add(1)
			go deploy("app2", &wg)
			wg.Wait()

			fmt.Println("apps: successfully deployed: app1, app2")
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

func deploy(app string, wg *sync.WaitGroup) {
	defer wg.Done()
	bar := uiprogress.AddBar(len(steps)).AppendCompleted().PrependElapsed()
	bar.Width = 50

	// prepend the deploy step to the bar
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(app+": "+steps[b.Current()-1], 22)
	})

	rand.Seed(500)
	for bar.Incr() {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(2000)))
	}
}
