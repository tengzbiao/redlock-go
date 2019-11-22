package redlock_go

import (
	"testing"
	"time"
)

func TestRedLock(t *testing.T) {
	servers := []string{"redis://127.0.0.1:6379/0"}
	lock := NewRedLock(servers)
	ret := lock.Lock("teacher:1", 60)
	defer lock.Unlock(ret)

	time.Sleep(2 * time.Second)

	t.Log(ret)
}

func TestRedLock_actAsync(t *testing.T) {
	servers := []string{"redis://127.0.0.1:6379/0"}
	lock := NewRedLock(servers)

	n1 := lock.actAsync(func(pool Pool) bool {
		time.Sleep(1 * time.Second)
		return false
	})
	t.Log(n1 == 0)

	n2 := lock.actAsync(func(pool Pool) bool {
		time.Sleep(1 * time.Second)
		return true
	})
	t.Log(n2 == 0)
}
