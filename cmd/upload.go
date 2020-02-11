package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shipengqi/lighting-i/pkg/filelock"
	"github.com/shipengqi/lighting-i/pkg/log"
	"github.com/shipengqi/lighting-i/pkg/utils"
)

func addUploadFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&Conf.Registry, "registry", "r", "https://registry-1.docker.io", "The host of the registry.")
	flagSet.StringVarP(&Conf.Org, "organization", "o", "", "Organization name of the images.")
	flagSet.StringVarP(&Conf.Dir, "dir", "d", "", "Images tar directory path.")
	flagSet.StringVarP(&Conf.User, "user", "u", "", "Registry account username.")
	flagSet.StringVarP(&Conf.Password, "pass", "p", "", "Registry account password.")
	flagSet.IntVarP(&Conf.RetryTimes, "retry", "t", 0, "The retry times when the image download fails.")
	flagSet.BoolVarP(&Conf.AutoConfirm, "yes", "y", false, "Answer yes for any confirmations.")
	flagSet.BoolVarP(&Conf.Force, "force", "f", false, "If true, ignore the process lock.")
}

func uploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:	_defaultUploadCommand,
		Aliases: []string{_defaultUploadAlias},
		Short:	"Upload docker images.",
		PreRun: func(cmd *cobra.Command, args []string) {
			if Conf.Dir == "" {
				log.Warn("Images tar directory path is required, pleased use '--dir' or '-d'.")
				filelock.UnLock(_defaultUploadLockFile)
				os.Exit(1)
			}
			if !utils.PathIsExist(Conf.Dir) {
				log.Warn("Images tar directory path is invalid.")
				filelock.UnLock(_defaultUploadLockFile)
				os.Exit(1)
			}
			if !utils.PathIsExist(filepath.Join(Conf.Dir, _defaultDownloadManifest)) {
				log.Warn("'images.download.manifest' file is invalid.")
				filelock.UnLock(_defaultUploadLockFile)
				os.Exit(1)
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

func getImagesDownloadManifest(manifest string) ([]DownloadManifest, error) {
	var dm []DownloadManifest
	data, err := ioutil.ReadFile(manifest)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %v", err)
	}
	err = json.Unmarshal(data, dm)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}
	return dm, nil
}

func uploadImages() {

}

func uploadLayersOfImage() {

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