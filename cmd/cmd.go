package cmd

import (
	"fmt"
	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/spf13/cobra"
	"math/rand"
	"sync"
	"time"
)

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

func NewLightingCommand() *cobra.Command {
	lightingCmd := &cobra.Command{
		Use:	"lighting",
		Short:	"suite-installer internal API server.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {

		},
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

	lightingCmd.ResetFlags()
	// Add sub commands
	lightingCmd.AddCommand(downloadCommand())
	lightingCmd.AddCommand(uploadCommand())
	return lightingCmd
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