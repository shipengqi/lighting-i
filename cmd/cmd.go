package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/spf13/cobra"

	"github.com/shipengqi/lighting-i/pkg/docker/registry/client"
	"github.com/shipengqi/lighting-i/pkg/log"
)

var (
	_defaultRootCommand      = "lighting"
	_defaultDownloadCommand  = "download"
	_defaultDownloadAlias    = "down"
	_defaultUploadCommand    = "upload"
	_defaultUploadAlias      = "up"
	_defaultBaseDir          = "/var/opt/lighting"
	_defaultImageSet         = _defaultBaseDir + "/image_set.yaml"
	_defaultImagesDir        = _defaultBaseDir + "/offline"
	_defaultDownloadLockFile = _defaultBaseDir + "/images.download.lock"
	_defaultUploadLockFile   = _defaultBaseDir + "/images.upload.lock"
	_defaultManifestJson     = "manifest.json"
	_defaultDownloadManifest = "images.download.manifest"
	_defaultUploadManifest   = "images.upload.manifest"
	_defaultDownloadLog      = "images.download.log"
	_defaultUploadLog        = "images.upload.log"
)

var Conf Config
var ImageDateFolderPath string
var LogFilePath string
var c *client.Client

type Config struct {
	AutoConfirm bool
	Dir         string
	User        string
	Password    string
	RetryTimes  int
	Registry    string
	Key         string
	Force       bool

	Org         string
	ImagesSet   string
	Overwrite   bool
}

func NewLightingCommand() *cobra.Command {
	lightingCmd := &cobra.Command{
		Use:   _defaultRootCommand,
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

	// Reset Flags
	lightingCmd.ResetFlags()

	// Disable commands sorting
	cobra.EnableCommandSorting = false
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
			log.Warn("[SIGNAL] Catch SIGINT")
			exitCh <- 0

		case syscall.SIGTERM: // kill -SIGTERM XXXX
			log.Warn("[SIGNAL] Catch SIGTERM")
			exitCh <- 1

		case syscall.SIGQUIT: // kill -SIGQUIT XXXX
			log.Warn("[SIGNAL] Catch SIGQUIT")
			exitCh <- 0
		default:
			return
		}
	}
}

func initClient() error {
	c = client.New()
	c.SetHostURL(Conf.Registry)
	c.SetSecureSkip(true)
	c.SetUsername(Conf.User)
	c.SetPassword(Conf.Password)
	c.SetRetryCount(Conf.RetryTimes)
	c.SetRetryMaxWaitTime(time.Second * 5)

	log.Infof("Ping %s ...", c.HostURL)
	if err := c.Ping(); err != nil {
		log.Errorf("ping registry %v.", err)
		return err
	}
	log.Infof("Ping %s OK", c.HostURL)
	return nil
}

func initDir(dirPath string) (string, error) {
	folderPath := filepath.Join(dirPath, time.Now().Format("20060102150405"))
	if err := os.MkdirAll(folderPath, 777); err != nil {
		return "", err
	}
	return folderPath, nil
}

func addProgressBar(total int, image client.ImageRepo) *uiprogress.Bar {
	title := fmt.Sprintf("%s:%s", strings.Split(image.Name, "/")[1], image.Tag)
	bar := uiprogress.AddBar(total).AppendCompleted().AppendElapsed()
	bar.Width = _defaultProgressWidth
	// prepend the deploy step to the bar
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return strutil.Resize(title, uint(_defaultProgressTitleWidth))
	})
	return bar
}