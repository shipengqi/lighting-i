package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/docker/registry/client"
	"github.com/shipengqi/lighting-i/pkg/filelock"
	"github.com/shipengqi/lighting-i/pkg/log"
	"github.com/shipengqi/lighting-i/pkg/utils"
)

type UploadConfig struct {
	Dir         string
	User        string
	Password    string
	RetryTimes  int
	Registry    string
	Force       bool
	Org         string
	Overwrite   bool
}

type UploadManifest struct {
	Layers []LayerResponse
	Image  client.ImageRepo
}

var uploadConfig UploadConfig

func addUploadFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&uploadConfig.Registry, "registry", "r", "https://registry-1.docker.io", "The host of the registry.")
	flagSet.StringVarP(&uploadConfig.Org, "organization", "o", "", "Organization name of the images.")
	flagSet.StringVarP(&uploadConfig.Dir, "dir", "d", "", "Images tar directory path (required).")
	flagSet.StringVarP(&uploadConfig.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&uploadConfig.Password, "pass", "p", "", "Registry account password.")
	flagSet.IntVarP(&uploadConfig.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
	flagSet.BoolVarP(&uploadConfig.Force, "force", "f", false, "If true, ignore the process lock.")
	flagSet.BoolVarP(&uploadConfig.Overwrite, "overwrite", "w", false, "If true, overwrite the existing images on the registry.")
}

func uploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:	_defaultUploadCommand,
		Aliases: []string{_defaultUploadAlias},
		Short:	"Upload docker images.",
		PreRun: func(cmd *cobra.Command, args []string) {
			// Init Conf
			Conf.RetryTimes = uploadConfig.RetryTimes
			Conf.Registry = uploadConfig.Registry
			Conf.User = uploadConfig.User
			Conf.Password = uploadConfig.Password

			if uploadConfig.Dir == "" {
				fmt.Println("Images tar directory path is required, pleased use '--dir' or '-d'.")
				os.Exit(1)
			}

			if !utils.PathIsExist(uploadConfig.Dir) {
				fmt.Println("Images tar directory path is invalid.")
				os.Exit(1)
			}

			if !utils.PathIsExist(filepath.Join(uploadConfig.Dir, _defaultDownloadManifest)) {
				fmt.Println("'images.download.manifest' file is invalid.")
				os.Exit(1)
			}

			ImageDateFolderPath = uploadConfig.Dir
			LogFilePath = filepath.Join(ImageDateFolderPath, _defaultUploadLog)
			log.Init(LogFilePath)

			if uploadConfig.Force {
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

			dm, err := getImagesDownloadManifest(filepath.Join(uploadConfig.Dir, _defaultDownloadManifest))
			if err != nil {
				log.Errorf("get manifest %v.", err)
				return
			}
			log.Debug("read download manifest", dm)
			log.Infof("Starting the upload the images to %s under %s ...", uploadConfig.Org, ImageDateFolderPath)

			completedc := make(chan int, 1)
			go uploadImages(dm, completedc)

			exitc := make(chan int, 1)
			go handleSignals(exitc)
			for {
				select {
				case failed := <-completedc:
					if failed < 1 {
						log.Infof("Successfully upload the images to %s under %s .", uploadConfig.Org, ImageDateFolderPath)
					} else {
						log.Errorf("Upload images with %d error(s).", failed)
					}

					log.Infof("You can refer to %s for more detail.", LogFilePath)
					filelock.UnLock(_defaultUploadLockFile)
					os.Exit(0)
				case code := <-exitc:
					filelock.UnLock(_defaultUploadLockFile)
					os.Exit(code)
				default:
					time.Sleep(10 * time.Millisecond)
				}
			}
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

func uploadImages(dm []DownloadManifest, completedc chan int) {
	var wg sync.WaitGroup
	var ums []*UploadManifest
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
	err := generateUploadManifest(ums)
	if err != nil {
		log.Errorf("generate manifest %v.", err)
	}
	uiprogress.Stop()
	failed := checkUploadBlobsResult(ums)
	completedc <- failed
}

func uploadLayersOfImage(m DownloadManifest, bar *uiprogress.Bar) *UploadManifest {
	var wg sync.WaitGroup
	um := &UploadManifest{Image: m.Image}
	if !uploadConfig.Overwrite && checkImagesTagIsExists(m.Image.Name, m.Image.Tag) {
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
		go func(l LayerResponse) {
			defer wg.Done()
			err := uploadBlobs(m.Image, l)
			log.Debugf("upload blobs %s of %s, status: %d, %s.", l.Target, m.Image.Name, err.Code, err.Message)
			um.Layers = append(um.Layers, LayerResponse{err, l.Digest, l.Target})
			bar.Incr()
		}(l)
	}
	wg.Wait()
	return um
}

func uploadBlobs(i client.ImageRepo, l LayerResponse) *client.Errno {
	res := c.StartUpload(i.Name)
	if res.Code != client.OK.Code {
		return res
	}
	uuid := res.Message
	res = c.PushBlobs(i.Name, l.Digest, uuid, l.Target)
	return res
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

func checkUploadBlobsResult(ums []*UploadManifest) int {
	var failed int
	for _, m := range ums {
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

func generateUploadManifest(ums []*UploadManifest) error {
	j, err := json.Marshal(ums)
	if err != nil {
		return fmt.Errorf("unmarshal %v", err)
	}
	err = ioutil.WriteFile(filepath.Join(ImageDateFolderPath, _defaultUploadManifest), j, 777)
	if err != nil {
		return fmt.Errorf("write %v", err)
	}
	return nil
}