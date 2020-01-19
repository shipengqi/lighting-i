package filelock

import (
	"fmt"
	"os"

	"github.com/shipengqi/lighting-i/pkg/utils"
)

func Check(name string) bool {
	return utils.PathIsExist(name)
}

func Lock(name string) error {
	if Check(name) {
		return fmt.Errorf("lock exists")
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

func UnLock(name string) error {
	err := os.Remove(name)
	if err != nil {
		return err
	}
	return nil
}