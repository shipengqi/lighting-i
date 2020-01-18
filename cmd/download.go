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

    "github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/docker/registry/client"
	"github.com/shipengqi/lighting-i/pkg/filelock"
	"github.com/shipengqi/lighting-i/pkg/images"
)

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



func addDownloadFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&Conf.ImagesSet, "image-set", "i", _defaultImageSet, "Images set file path.")
}

func downloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download docker images.",
		Run: func(cmd *cobra.Command, args []string) {
			defer filelock.UnLock(_defaultDownloadLockFile)
			c := client.New()
			c.SetHostURL(Conf.Registry)
			c.SetSecureSkip(true)
			c.SetUsername(Conf.User)
			c.SetPassword(Conf.Password)

			if err := c.Ping(); err != nil {
				fmt.Printf("ping registry %v.\n", err)
				return
			}

			imageSet, err := images.GetImagesFromSet(Conf.ImagesSet)
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
			err = ioutil.WriteFile(filepath.Join(ImageDateFolderPath, _defaultManifestJson), j, 777)
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
						fmt.Println("Download successfully.")
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
	addDownloadFlags(cmd.Flags())
	return cmd
}

func calculateRequiredLayers() {

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
			o := fmt.Sprintf("%s/%s.tar.gz", ImageDateFolderPath, strings.Split(l.Digest, ":")[1])
			err := c.GetBlobs(mr.ImageName, l.Digest, o)
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