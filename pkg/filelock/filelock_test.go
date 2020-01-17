package filelock

import "testing"

func TestLock(t *testing.T) {
	lockfile := "test.lock"
	UnLock(lockfile)
	t.Run("Lock success", func(t *testing.T) {
		err := Lock(lockfile)
		if err != nil {
			t.Fatal("Wanted true, got false")
		}
	})

	t.Run("Lock failed", func(t *testing.T) {
		err := Lock(lockfile)
		if err == nil {
			t.Fatal("Wanted false, got true")
		}
	})


}

func TestUnLock(t *testing.T) {
	lockfile := "test.lock"
	t.Run("UnLock success", func(t *testing.T) {
		err := UnLock(lockfile)
		if err != nil {
			t.Fatal("Wanted true, got false")
		}
	})

	t.Run("UnLock failed", func(t *testing.T) {
		err := UnLock(lockfile)
		if err == nil {
			t.Fatal("Wanted false, got true")
		}
	})
}