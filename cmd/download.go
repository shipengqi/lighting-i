package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/shipengqi/lighting-i/pkg/utils"
	"io/ioutil"
	"math"
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

var c *client.Client

var (
	_defaultProgressWidth      = 50
	_defaultProgressTitleWidth = 30
)

type ManifestResponse struct {
	Status   *client.Errno
	Manifest *client.Manifest
}

type LayerResponse struct {
	Status *client.Errno
	Target string
}

type RequiredLayer struct {
	Fetched bool
	Layer   client.Layer
	Image   client.ImageRepo
}

type DownloadManifest struct {
	Config LayerResponse
	Layers []LayerResponse
	Image  client.ImageRepo
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
			if !checkImageSet(Conf.ImagesSet) {
				fmt.Printf("%s is not exists.\n", Conf.ImagesSet)
				return
			}
			c = client.New()
			c.SetHostURL(Conf.Registry)
			c.SetSecureSkip(true)
			c.SetUsername(Conf.User)
			c.SetPassword(Conf.Password)
			c.SetRetryCount(Conf.RetryTimes)
			c.SetRetryMaxWaitTime(time.Second * 5)

			if err := c.Ping(); err != nil {
				fmt.Printf("ping registry %v.\n", err)
				return
			}

			imageSet, err := images.GetImagesFromSet(Conf.ImagesSet)
			if err != nil {
				fmt.Printf("get images %v.\n", err)
				return
			}
			if imageSet.OrgName == "" {
				imageSet.OrgName = "official library"
			}
			fmt.Printf("Starting the download of the %s ...\n", imageSet.OrgName)

			allManifest := fetchAllManifest(imageSet)
			err = generateManifestFile(allManifest)
			if err != nil {
				fmt.Printf("manifest file %v.\n", err)
				return
			}

			required, total := calculateRequiredLayers(allManifest)
			fmt.Println("Warning: Please make sure you have enough disk space for downloading images.")
			fmt.Printf("Total size of the images: %d MB.\n", total)

			completedc := make(chan int, 1)
			go downloadImages(allManifest, required, completedc)

			exitc := make(chan int, 1)
			go handleSignals(exitc)
			for {
				select {
				case <-completedc:
					fmt.Println("Download successfully.")
					filelock.UnLock(_defaultDownloadLockFile)
					os.Exit(0)
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

func checkImageSet(name string) bool {
	return utils.PathIsExist(name)
}

func generateManifestFile(manifests []ManifestResponse) error {
	j, err := json.Marshal(manifests)
	if err != nil {
		return fmt.Errorf("unmarshal %v", err)
	}
	err = ioutil.WriteFile(filepath.Join(ImageDateFolderPath, _defaultManifestJson), j, 777)
	if err != nil {
		return fmt.Errorf("write %v", err)
	}
	return nil
}

func generateDownloadManifest(dms []*DownloadManifest) error {
	j, err := json.Marshal(dms)
	if err != nil {
		return fmt.Errorf("unmarshal %v", err)
	}
	err = ioutil.WriteFile(filepath.Join(ImageDateFolderPath, _defaultDownloadManifest), j, 777)
	if err != nil {
		return fmt.Errorf("write %v", err)
	}
	return nil
}

func fetchAllManifest(imageSet *images.ImageSet) []ManifestResponse {
	var wg sync.WaitGroup
	var manifests []ManifestResponse
	wg.Add(len(imageSet.Images))
	for _, i := range imageSet.Images {
		go func(i string) {
			defer wg.Done()
			img := images.ParseImage(i, imageSet.OrgName)
			manifest, err := c.FetchManifest(img.Name, img.Tag)
			manifests = append(manifests, ManifestResponse{err, manifest})
		}(i)
	}
	wg.Wait()
	return manifests
}

func downloadImages(manifests []ManifestResponse, required *sync.Map, completedc chan int) {
	var wg sync.WaitGroup
	var dms []*DownloadManifest
	wg.Add(len(manifests))
	uiprogress.Start()
	for _, m := range manifests {
		bar := addProgressBar(len(m.Manifest.Layers), m.Manifest.Image)
		go func(m ManifestResponse, bar2 *uiprogress.Bar) {
			defer wg.Done()
			dm := fetchLayersOfManifest(m, required, bar2)
			dms = append(dms, dm)
		}(m, bar)
	}
	wg.Wait()
	err := generateDownloadManifest(dms)
	if err != nil {
		fmt.Printf("download manifest %v.\n", err)
	}
	uiprogress.Stop()
	completedc <- 1
}

func calculateRequiredLayers(manifests []ManifestResponse) (*sync.Map, int) {
	s := new(sync.Map)
	var totalSize int
	for _, m := range manifests {
		if m.Status.Code != client.OK.Code {
			continue
		}
		if len(m.Manifest.Layers) < 1 {
			continue
		}
		for _, l := range m.Manifest.Layers {
			totalSize += int(math.Ceil(float64(l.Size / 1024 / 1024)))
			s.LoadOrStore(l.Digest, RequiredLayer{false, l, m.Manifest.Image})
		}
	}
	return s, totalSize
}

func fetchConfigOfManifest(mr ManifestResponse) (*client.Errno, string) {
	target := fmt.Sprintf("%s/%s.json", ImageDateFolderPath, strings.Split(mr.Manifest.Config.Digest, ":")[1])
	err := c.FetchBlobs(mr.Manifest.Image.Name, mr.Manifest.Config.Digest, target)
	return err, target
}

func fetchLayersOfManifest(mr ManifestResponse, required *sync.Map, bar *uiprogress.Bar) *DownloadManifest {
	var wg sync.WaitGroup
	lm := &DownloadManifest{Image: mr.Manifest.Image}
	err, conf := fetchConfigOfManifest(mr)
	lm.Config = LayerResponse{err, conf}
	for _, l := range mr.Manifest.Layers {
		v, _ := required.Load(l.Digest)
		s, _ := v.(RequiredLayer)
		target := fmt.Sprintf("%s/%s.tar.gz", ImageDateFolderPath, strings.Split(l.Digest, ":")[1])
		if s.Fetched == true {
			lm.Layers = append(lm.Layers, LayerResponse{client.OK, target})
			bar.Incr()
			continue
		}
		wg.Add(1)
		go func(l client.Layer, t string) {
			defer wg.Done()
			err := c.FetchBlobs(mr.Manifest.Image.Name, l.Digest, t)
			lm.Layers = append(lm.Layers, LayerResponse{err, t})
			bar.Incr()
		}(l, target)
	}
	wg.Wait()
	return lm
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
