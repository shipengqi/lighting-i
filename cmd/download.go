package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/docker/registry/client"
	"github.com/shipengqi/lighting-i/pkg/filelock"
	"github.com/shipengqi/lighting-i/pkg/images"
	"github.com/shipengqi/lighting-i/pkg/log"
	"github.com/shipengqi/lighting-i/pkg/utils"
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

type ManifestCheckResult struct {
	Required  *sync.Map
	Failed    []ManifestResponse
	TotalSize int
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
	flagSet.StringVarP(&Conf.Registry, "registry", "r", "https://registry-1.docker.io", "The host of the registry.")
	flagSet.StringVarP(&Conf.ImagesSet, "image-set", "i", _defaultImageSet, "Images set file path.")
	flagSet.StringVarP(&Conf.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&Conf.Password, "pass", "p", "", "Registry account password.")
	flagSet.IntVarP(&Conf.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
	flagSet.StringVarP(&Conf.Dir, "dir", "d", _defaultImagesDir, "Images tar directory path.")
	flagSet.BoolVarP(&Conf.AutoConfirm, "yes", "y", false, "Answer yes for any confirmations.")
	flagSet.BoolVarP(&Conf.Force, "force", "f", false, "If true, ignore the process lock.")
}

func downloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   _defaultDownloadCommand,
		Aliases: []string{_defaultDownloadAlias},
		Short: "Download docker images.",
		Run: func(cmd *cobra.Command, args []string) {
			defer func() {
				log.Infof("You can refer to %s for more detail.", LogFilePath)
				filelock.UnLock(_defaultDownloadLockFile)
			}()
			if !checkImageSet(Conf.ImagesSet) {
				log.Errorf("%s is not exists.", Conf.ImagesSet)
				return
			}
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
				return
			}
			log.Infof("Ping %s OK", c.HostURL)
			imageSet, err := images.GetImagesFromSet(Conf.ImagesSet)
			if err != nil {
				log.Errorf("get images %v.", err)
				return
			}
			log.Debug("read image set", imageSet)
			if imageSet.OrgName == "" {
				imageSet.OrgName = "official library"
			}
			log.Infof("Starting the download of the %s ...", imageSet.OrgName)

			allManifest := fetchAllManifest(imageSet)
			log.Debug("fetch manifest", allManifest)
			mcr := checkFetchManifestResult(allManifest)
			if len(mcr.Failed) > 0 {
				log.Errorf("Fetch images manifest with errors.")
				return
			}
			err = generateManifestFile(allManifest)
			if err != nil {
				log.Errorf("manifest file %v.", err)
				return
			}

			log.Info("Warning: Please make sure you have enough disk space for downloading images.")
			log.Infof("Total size of the images: %d MB.", mcr.TotalSize)

			completedc := make(chan int, 1)
			go downloadImages(allManifest, mcr.Required, completedc)

			exitc := make(chan int, 1)
			go handleSignals(exitc)
			for {
				select {
				case ccode := <-completedc:
					if ccode < 1 {
						log.Infof("Successfully downloaded the images to %s.", ImageDateFolderPath)
					} else {
						log.Errorf("Download images with errors.")
					}

					log.Infof("You can refer to %s for more detail.", LogFilePath)
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
			log.Debugf("fetch manifest: %s:%s, status: %d, %s.", img.Name, img.Tag, err.Code, err.Message)
			manifests = append(manifests, ManifestResponse{err, manifest})
		}(i)
	}
	wg.Wait()
	return manifests
}

func downloadImages(manifests []ManifestResponse, required *sync.Map, completedc chan int) {
	var wg sync.WaitGroup
	var dms []*DownloadManifest
	log.Debugf("download blobs with %d goroutines.", len(manifests))
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
	log.Debug("download blobs completed.")
	err := generateDownloadManifest(dms)
	if err != nil {
		log.Errorf("download manifest %v.", err)
	}
	uiprogress.Stop()
	fbr := checkFetchBlobsResult(dms)
	if fbr > 0 {
		completedc <- 1
		return
	}
	completedc <- 0
}

func fetchConfigOfManifest(mr ManifestResponse) (string, *client.Errno) {
	target := fmt.Sprintf("%s/%s.json", ImageDateFolderPath, strings.Split(mr.Manifest.Config.Digest, ":")[1])
	err := c.FetchBlobs(mr.Manifest.Image.Name, mr.Manifest.Config.Digest, target)
	return target, err
}

func fetchLayersOfManifest(mr ManifestResponse, required *sync.Map, bar *uiprogress.Bar) *DownloadManifest {
	var wg sync.WaitGroup
	log.Debugf("fetch config of manifest: %s:%s.", mr.Manifest.Image.Name, mr.Manifest.Image.Tag)
	lm := &DownloadManifest{Image: mr.Manifest.Image}
	conf, err := fetchConfigOfManifest(mr)
	log.Debugf("fetch config of manifest: %s:%s, status: %d, %s.", mr.Manifest.Image.Name, mr.Manifest.Image.Tag, err.Code, err.Message)
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
			log.Debugf("fetch blobs %s of %s, status: %d, %s.", l.Digest, mr.Manifest.Image.Name, err.Code, err.Message)
			lm.Layers = append(lm.Layers, LayerResponse{err, t})
			bar.Incr()
		}(l, target)
	}
	wg.Wait()
	return lm
}

func checkFetchManifestResult(manifests []ManifestResponse) *ManifestCheckResult {
	mcr := &ManifestCheckResult{Required: new(sync.Map)}
	for _, m := range manifests {
		if m.Status.Code != client.OK.Code {
			mcr.Failed = append(mcr.Failed, m)
			continue
		}
		if len(m.Manifest.Layers) < 1 {
			continue
		}
		for _, l := range m.Manifest.Layers {
			mcr.TotalSize += int(math.Ceil(float64(l.Size / 1024 / 1024)))
			mcr.Required.LoadOrStore(l.Digest, RequiredLayer{false, l, m.Manifest.Image})
		}
	}
	return mcr
}

func checkFetchBlobsResult(dms []*DownloadManifest) int {
	var failed int
	for _, m := range dms {
		if len(m.Layers) < 1 {
			continue
		}
		for _, l := range m.Layers {
			if l.Status.Code != client.OK.Code {
				failed ++
			}
		}
	}
	return failed
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
