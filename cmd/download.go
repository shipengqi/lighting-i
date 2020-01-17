package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shipengqi/lighting-i/pkg/docker/registry/client"
	"github.com/shipengqi/lighting-i/pkg/filelock"
	"github.com/shipengqi/lighting-i/pkg/images"

    "github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type DownloadConfig struct {
	AutoConfirm bool
	Dir         string
	ImagesSet   string
	User        string
	Password    string
	RetryTimes  int
	Registry    string
	Key         string
}

type ManifestRes struct {
	OK        bool
	Message   string
	ImageName string
	ImageTag  string
	Manifest  *client.Manifest
}

type LayerRes struct {
	OK        bool
	Message   string
	ImageName string
	ImageTag  string
}

var imageFolderPath string

func addDownloadFlags(flagSet *pflag.FlagSet, conf *DownloadConfig) {
	flagSet.StringVarP(&conf.Registry, "registry", "r", "https://registry-1.docker.io", "The host of the registry.")
	flagSet.StringVarP(&conf.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&conf.Password, "pass", "p", "", "Registry account password.")
	flagSet.StringVar(&conf.Key, "key", "", "Key file registry account.")
	flagSet.IntVarP(&conf.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
	flagSet.StringVarP(&conf.ImagesSet, "image-set", "i", _defaultImageSet, "Images set file path.")
	flagSet.StringVarP(&conf.Dir, "dir", "d", _defaultImagesDir, "Images tar directory path.")
	flagSet.BoolVarP(&conf.AutoConfirm, "yes", "y", false, "Answer yes for any confirmations.")
}

func downloadCommand() *cobra.Command {
	var conf = &DownloadConfig{}
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download docker images.",
		PreRun: func(cmd *cobra.Command, args []string) {
			// Create required dir and create download directory by date
			folderPath, err := createDir(conf.Dir)
			if err != nil {
				fmt.Printf("mkdir %v", err)
				os.Exit(1)
			}
			imageFolderPath = folderPath

			if err := filelock.Lock(_defaultDownloadLockFile); err != nil {
				fmt.Println("Error: one instance is already running and only one instance is allowed at a time.")
				fmt.Println("Check to see if another instance is running.")
				fmt.Printf("If the instance stops running, delete %s file.\n", _defaultDownloadLockFile)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			defer filelock.UnLock(_defaultDownloadLockFile)
			c := client.New(conf.User, conf.Password, conf.Registry)
			// Ping get registry auth info
			if err := c.Ping(); err != nil {
				fmt.Printf("ping registry %v.\n", err)
				return
			}
			imageSet, err := images.GetImagesFromSet(conf.ImagesSet)
			if err != nil {
				fmt.Printf("get images %v.\n", err)
				return
			}
			allManifest := getAllManifest(c, imageSet)
			j, err := json.Marshal(allManifest)
			if err != nil {
				fmt.Printf("manifest unmarshal %v.\n", err)
				return
			}
			err = ioutil.WriteFile(filepath.Join(imageFolderPath, _defaultManifestJson), j, 777)
			if err != nil {
				fmt.Printf("write manifest %v.\n", err)
				return
			}
			fmt.Println("start download images.")
			downloadc := make(chan int, 1)
			downloadCount := len(allManifest)
			uiprogress.Start()

			for _, m := range allManifest {
				bar := uiprogress.AddBar(len(m.Manifest.Layers)).AppendCompleted().PrependElapsed()
				go func(m ManifestRes, bar2 *uiprogress.Bar) {
					getLayersByImageRepo(c, m, downloadc, bar2)
				}(m, bar)
			}


			exitc := make(chan int, 1)
			go handleSignals(exitc)
			for {
				select {
				case <-downloadc:
					downloadCount --
					if downloadCount <= 0 {
						fmt.Println("download successfully.")
						filelock.UnLock(_defaultDownloadLockFile)
						os.Exit(0)
					}
				case code := <-exitc:
					filelock.UnLock(_defaultDownloadLockFile)
					os.Exit(code)
				default:
					time.Sleep(10 * time.Millisecond)
				}
			}
		},
	}
	cmd.Flags().SortFlags = false
	addDownloadFlags(cmd.Flags(), conf)
	return cmd
}

func getAllManifest(c *client.Client, imageSet *images.ImageSet) []ManifestRes {
	var wg sync.WaitGroup
	var manifests []ManifestRes
	wg.Add(len(imageSet.Images))
	for _, i := range imageSet.Images {
		go func(i string) {
			defer wg.Done()
			img := images.ParseImage(i, imageSet.OrgName)
			manifest, err := c.GetManifest(img.Name, img.Tag)
			if err != nil {
				manifests = append(manifests, ManifestRes{
					OK:        false,
					Message:   err.Error(),
					ImageName: img.Name,
					ImageTag: img.Tag,
				})
			} else {
				manifests = append(manifests, ManifestRes{
					OK:        true,
					ImageName: img.Name,
					ImageTag: img.Tag,
					Manifest:  manifest,
				})
			}
		}(i)
	}
	wg.Wait()
	return manifests
}

func getLayersByImageRepo(c *client.Client, mr ManifestRes, downloadc chan int, bar *uiprogress.Bar) []LayerRes {
	if len(mr.Manifest.Layers) < 1 {
		return nil
	}
	var wg sync.WaitGroup
	var layers []LayerRes
	wg.Add(len(mr.Manifest.Layers))
	for _, l := range mr.Manifest.Layers {
		go func(l client.Layer) {
			defer wg.Done()
			o := fmt.Sprintf("%s/%s.tar.gz", imageFolderPath, strings.Split(l.Digest, ":")[1])
			err := c.GetLayerBlobs(mr.ImageName, l.Digest, o)
			if err != nil {
				layers = append(layers, LayerRes{
					OK:        false,
					Message:   err.Error(),
					ImageName: mr.ImageName,
					ImageTag: mr.ImageTag,
				})
			} else {
				layers = append(layers, LayerRes{
					OK:        true,
					ImageName: mr.ImageName,
					ImageTag: mr.ImageTag,
				})
			}
			bar.Incr()
		}(l)
	}
	wg.Wait()
	downloadc <- 1
	return layers
}


func createDir(dirPath string) (string, error) {
	folderPath := filepath.Join(dirPath, time.Now().Format("20060102"))
	if err := os.MkdirAll(folderPath, 777); err != nil {
		return "", err
	}
	return folderPath, nil
}