package filelock

import "os"

func Check(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func Lock(name string) bool {
	if Check(name) {
		return false
	}
	_, err := os.Create(name)
	if err != nil {
		return false
	}
	return true
}

func UnLock(name string) bool {
	err := os.Remove(name)
	if err != nil {
		return false
	}
	return true
}