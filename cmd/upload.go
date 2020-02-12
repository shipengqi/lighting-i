package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/shipengqi/lighting-i/pkg/docker/registry/client"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/filelock"
	"github.com/shipengqi/lighting-i/pkg/log"
	"github.com/shipengqi/lighting-i/pkg/utils"
)

type UploadManifest struct {
	Layers []LayerResponse
	Image  client.ImageRepo
}

func addUploadFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&Conf.Registry, "registry", "r", "https://registry-1.docker.io", "The host of the registry.")
	flagSet.StringVarP(&Conf.Org, "organization", "o", "", "Organization name of the images.")
	flagSet.StringVarP(&Conf.Dir, "dir", "d", "", "Images tar directory path (required).")
	flagSet.StringVarP(&Conf.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&Conf.Password, "pass", "p", "", "Registry account password.")
	flagSet.IntVarP(&Conf.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
	flagSet.BoolVarP(&Conf.AutoConfirm, "yes", "y", false, "Answer yes for any confirmations.")
	flagSet.BoolVarP(&Conf.Force, "force", "f", false, "If true, ignore the process lock.")
	flagSet.BoolVarP(&Conf.Overwrite, "overwrite", "w", false, "If true, overwrite the existing images on the registry.")
}

func uploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:	_defaultUploadCommand,
		Aliases: []string{_defaultUploadAlias},
		Short:	"Upload docker images.",
		PreRun: func(cmd *cobra.Command, args []string) {
			if Conf.Dir == "" {
				fmt.Println("Images tar directory path is required, pleased use '--dir' or '-d'.")
				os.Exit(1)
			}

			if !utils.PathIsExist(Conf.Dir) {
				fmt.Println("Images tar directory path is invalid.")
				os.Exit(1)
			}

			if !utils.PathIsExist(filepath.Join(Conf.Dir, _defaultDownloadManifest)) {
				fmt.Println("'images.download.manifest' file is invalid.")
				os.Exit(1)
			}

			ImageDateFolderPath = Conf.Dir
			LogFilePath = filepath.Join(ImageDateFolderPath, _defaultUploadLog)
			log.Init(LogFilePath)

			if Conf.Force {
				return
			}

			if err := filelock.Lock(_defaultUploadLockFile); err != nil {
				log.Error("One instance is already running and only one instance is allowed at a time.")
				log.Error("Check to see if another instance is running.")
				log.Fatalf("If the instance stops running, delete %s file.\n", _defaultUploadLockFile)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			defer func() {
				log.Infof("You can refer to %s for more detail.", LogFilePath)
				filelock.UnLock(_defaultUploadLockFile)
			}()

			err := initClient()
			if err != nil {
				log.Errorf("init client %v.", err)
				return
			}

			dm, err := getImagesDownloadManifest(filepath.Join(Conf.Dir, _defaultDownloadManifest))
			if err != nil {
				log.Errorf("get manifest %v.", err)
				return
			}
			log.Debug(dm)
			
		},
	}
	cmd.Flags().SortFlags = false
	addUploadFlags(cmd.Flags())
	return cmd
}

func getImagesDownloadManifest(file string) ([]DownloadManifest, error) {
	var dm []DownloadManifest
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %v", err)
	}
	err = json.Unmarshal(data, dm)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}
	return dm, nil
}

func uploadImages(dm []DownloadManifest, required *sync.Map, completedc chan int) {
	var wg sync.WaitGroup
	log.Debugf("upload images with %d goroutines.", len(dm))
	wg.Add(len(dm))
	uiprogress.Start()
	for _, m := range dm {
		bar := addProgressBar(len(m.Layers), m.Image)
		go func(m DownloadManifest, bar2 *uiprogress.Bar) {
			defer wg.Done()
			uploadLayersOfImage(m, bar2)
		}(m, bar)
	}
	wg.Wait()
	log.Debug("upload images completed.")
	err := generateDownloadManifest(dms)
	if err != nil {
		log.Errorf("generate manifest %v.", err)
	}
	uiprogress.Stop()
	fbr := checkFetchBlobsResult(dms)
	completedc <- fbr
}

func uploadLayersOfImage(m DownloadManifest, bar *uiprogress.Bar) *UploadManifest {
	var wg sync.WaitGroup
	um := &UploadManifest{Image: m.Image}
	if !Conf.Overwrite && checkImagesTagIsExists(m.Image.Name, m.Image.Tag) {
		_ = bar.Set(bar.Total)
		return um
	}
	for _, l := range m.Layers {
		if checkImagesLayerIsExists(m.Image.Name, l.Digest) {
			um.Layers = append(um.Layers, LayerResponse{client.OK, l.Digest,l.Target})
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

func uploadBlobs(m DownloadManifest) error {
	result := c.StartUpload(m.Image.Name)
	if result.Code != client.OK.Code {
		return result
	}

	return nil
}

func checkImagesTagIsExists(name, tag string) bool {
	list, err := c.ListImageTags(name)
	if err != nil {
		return false
	}
	for _, t := range list.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func checkImagesLayerIsExists(name, digest string) bool {
	err := c.CheckBlobs(name, digest)
	if err.Code == client.OK.Code {
		return true
	}
	return false
}