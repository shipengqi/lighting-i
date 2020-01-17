package filelock

import (
	"fmt"
	"os"
)

func Check(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
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