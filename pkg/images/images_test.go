package images

import (
	"strings"
	"testing"
)

func TestGetImagesFromConfig(t *testing.T) {
	t.Run("Got 31 images", func(t *testing.T) {
		images, _ := GetImagesFromConfig("../../images.yaml")
		want := 30
		if want != len(images.Images) {
			t.Fatalf("Wanted %d, got %v", want, len(images.Images))
		}
	})

    t.Run("Got error", func(t *testing.T) {
		_, err := GetImagesFromConfig("./images.yaml")
		want := "read images config"
		if !strings.Contains(err.Error(), "read images config") {
			t.Fatalf("Wanted %v, got %v", want, err)
		}
	})
}