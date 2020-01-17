package images

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	_defaultImageTag = "latest"
	_defaultOrgName  = "library"
)

type ImageSet struct {
	OrgName string   `yaml:"org_name"`
	Version string   `yaml:"version"`
	Images  []string `yaml:"images"`
}

type Image struct {
	Name string
	Tag  string
}

func GetImagesFromSet(set string) (*ImageSet, error) {
	imageSet := &ImageSet{}
	data, err := ioutil.ReadFile(set)
	if err != nil {
		return nil, fmt.Errorf("read images set: %v", err)
	}
	err = yaml.Unmarshal(data, imageSet)
	if err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %v", err)
	}
	return imageSet, nil
}

func ParseImage(image, org string) Image {
	if len(image) < 1 {
		return Image{}
	}
	name, tag := GetImageNameAndTag(image)
	if org == "" {
		org = _defaultOrgName
	}
	return Image{Name: fmt.Sprintf("%s/%s", org, name), Tag: tag}
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