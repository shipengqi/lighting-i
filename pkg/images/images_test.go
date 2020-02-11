package images

import (
	"strings"
	"testing"
)

func TestGetImagesFromConfig(t *testing.T) {
	t.Run("Got 31 images", func(t *testing.T) {
		images, _ := GetImagesFromSet("../../image_set.yaml")
		want := 28
		if want != len(images.Images) {
			t.Fatalf("Wanted %d, got %v", want, len(images.Images))
		}
	})

    t.Run("Got error", func(t *testing.T) {
		_, err := GetImagesFromSet("./image_set.yaml")
		want := "read images config"
		if !strings.Contains(err.Error(), "read images config") {
			t.Fatalf("Wanted %v, got %v", want, err)
		}
	})
}

func TestParseImage(t *testing.T) {
	t.Run("Parse image name addnode", func(t *testing.T) {
		image := ParseImage("addnode:1.5.0-002", "")
		want := Image{"library/addnode", "1.5.0-002"}
		if want != image {
			t.Fatalf("Wanted %v, got %v", want, image)
		}
	})

	t.Run("Parse image tag latest", func(t *testing.T) {
		image := ParseImage("addnode", "")
		want := Image{"library/addnode", "latest"}
		if want != image {
			t.Fatalf("Wanted %v, got %v", want, image)
		}
	})

	t.Run("Parse image empty", func(t *testing.T) {
		image := ParseImage("", "")
		want := Image{"", ""}
		if want != image {
			t.Fatalf("Wanted %v, got %v", want, image)
		}
	})
}

func TestReplaceImageOrg(t *testing.T) {
	t.Run("Replace image org to test", func(t *testing.T) {
		want := "test/itom-demo-core-tech-config"
		ns, err := ReplaceImageOrg("shipengqi/itom-demo-core-tech-config", "test")
		if err != nil {
			t.Fatalf("Wanted %v, got %v", want, err)
		}
		if want != ns {
			t.Fatalf("Wanted %v, got %v", want, ns)
		}
	})

	t.Run("Replace image org empty", func(t *testing.T) {
		want := "shipengqi/itom-demo-core-tech-config"
		ns, err := ReplaceImageOrg("shipengqi/itom-demo-core-tech-config", "")
		if err != nil {
			t.Fatalf("Wanted %v, got %v", want, err)
		}
		if want != ns {
			t.Fatalf("Wanted %v, got %v", want, ns)
		}
	})

	t.Run("Replace image name empty", func(t *testing.T) {
		want := ""
		ns, err := ReplaceImageOrg("", "")
		if err != nil {
			t.Fatalf("Wanted %v, got %v", want, err)
		}
		if want != ns {
			t.Fatalf("Wanted %v, got %v", want, ns)
		}
	})
}