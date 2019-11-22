# redlock-go
golang redis 分布式锁


### install
go get github.com/tengzbiao/redlock-go

### useage
```
servers := []string{"redis://123456@127.0.0.1:6379/0"}
lock := NewRedLock(servers)
// 加锁5秒
ret := lock.Lock("teacher:1", 5)
// 释放锁
defer lock.Unlock(ret)

// do your work
....
```
