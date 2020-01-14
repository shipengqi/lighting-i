package images

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Images struct {
	Images []string `yaml:"images"`
}

func GetImagesFromConfig(config string) (*Images, error) {
	images := &Images{}
	data, err := ioutil.ReadFile(config)
	if err != nil {
		return nil, fmt.Errorf("read images config: %v", err)
	}
	err = yaml.Unmarshal(data, images)
	if err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %v", err)
	}
	return images, nil
}

func GetImageTag(image string) string {
	return ""
}

func GetImageName(image string) string {
	return ""
}