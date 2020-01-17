package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/filelock"
)

var (
	_defaultBaseDir          = "/var/opt/lighting"
	_defaultImageSet         = _defaultBaseDir + "/image_set.yaml"
	_defaultImagesDir        = _defaultBaseDir + "/offline"
	_defaultDownloadLockFile = _defaultBaseDir + "/images.download.lock"
	_defaultUploadLockFile   = _defaultBaseDir + "/images.upload.lock"
	_defaultManifestJson     = "manifest.json"
)

var Conf Config
var ImageDateFolderPath string

type Config struct {
	AutoConfirm bool
	Org         string
	Dir         string
	User        string
	Password    string
	RetryTimes  int
	Registry    string
	Key         string
	Force       bool

	ImagesSet   string
}

func addLightingFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&Conf.Registry, "registry", "r", "https://registry-1.docker.io", "The host of the registry.")
	flagSet.StringVarP(&Conf.Org, "organization", "o", "", "Organization name of the images.")
	flagSet.StringVarP(&Conf.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&Conf.Password, "pass", "p", "", "Registry account password.")
	flagSet.StringVar(&Conf.Key, "key", "", "Key file registry account.")
	flagSet.IntVarP(&Conf.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
	flagSet.StringVarP(&Conf.Dir, "dir", "d", _defaultImagesDir, "Images tar directory path.")
	flagSet.BoolVarP(&Conf.AutoConfirm, "yes", "y", false, "Answer yes for any confirmations.")
	flagSet.BoolVarP(&Conf.Force, "force", "f", false, "If true, ignore the process lock.")
}


func NewLightingCommand() *cobra.Command {
	lightingCmd := &cobra.Command{
		Use:   "lighting",
		Short: "lighting is used to bulk download or upload docker images. It's much faster than 'docker pull'",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
				_ = cmd.Help()
				os.Exit(0)
			}

			// Create required dir and create download directory by date
			folderPath, err := initDir(Conf.Dir)
			if err != nil {
				fmt.Printf("mkdir %v", err)
				os.Exit(1)
			}
			ImageDateFolderPath = folderPath

			if Conf.Force {
				return
			}
			lockName := _defaultDownloadLockFile
			if cmd.Name() == "upload" {
				lockName = _defaultUploadLockFile
			}
			if err := filelock.Lock(lockName); err != nil {
				fmt.Println("Error: one instance is already running and only one instance is allowed at a time.")
				fmt.Println("Check to see if another instance is running.")
				fmt.Printf("If the instance stops running, delete %s file.\n", lockName)
				os.Exit(1)
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
	addLightingFlags(lightingCmd.PersistentFlags())
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

func initDir(dirPath string) (string, error) {
	folderPath := filepath.Join(dirPath, time.Now().Format("20060102"))
	if err := os.MkdirAll(folderPath, 777); err != nil {
		return "", err
	}
	return folderPath, nil
}