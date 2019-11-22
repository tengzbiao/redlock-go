package redlock_go

import (
	"testing"
)

func Test_newPools(t *testing.T) {
	servers := []string{
		"redis://123456@127.0.0.1:6379/0",
		"redis://123456@127.0.0.1:6380/1",
	}
	newPools(servers)
}