package images

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	_defaultImageTag = "latest"
)

type ImageConf struct {
	Images []string `yaml:"images"`
}

type Image struct {
	Name       string
	Tag        string
}

func GetImagesFromConfig(config string) (*ImageConf, error) {
	images := &ImageConf{}
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

func ParseImage(image string) Image {
	if len(image) < 1 {
		return Image{}
	}
	name, tag := GetImageNameAndTag(image)
	return Image{Name: name, Tag: tag}
}

func GetImageNameAndTag(image string) (string, string) {
	if len(image) < 1 {
		return "", ""
	}
	s := strings.Split(image, ":")
	if len(s) < 1 {
		return "", ""
	}

	if len(s) < 2 {
		return s[0], _defaultImageTag
	}

	return s[0], s[1]
}