package redlock_go

import (
	"errors"
	"math/rand"
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedLock struct {
	retryDelay int64   // 重试延迟(毫秒）
	retryCount int     // 重试次数
	clockDrift float64 // 时钟偏移
	quorum     int     // 正常工作的节点阈值
	pools      []Pool
}

func NewRedLock(servers []string) *RedLock {
	return &RedLock{
		retryDelay: 200,
		retryCount: 10,
		clockDrift: 0.01,
		quorum:     len(servers)/2 + 1,
		pools:      newPools(servers),
	}
}

type LockRet struct {
	Resource string // 锁的key
	Token    string // 锁的value
	State    bool   // 锁状态
}

// 加锁
func (r *RedLock) Lock(resource string, ttl int) (lockRet LockRet) {
	lockRet.Resource = resource
	lockRet.Token = uniqId()
	// ttl单位为秒
	for {
		if r.retryCount == 0 {
			return lockRet
		}

		startTime := time.Now().UnixNano() / 1e6

		n := r.actAsync(func(pool Pool) bool {
			return r.lock(pool, lockRet, ttl) == nil
		})
		// Add 2 milliseconds to the drift to account for Redis expires
		// precision, which is 1 millisecond, plus 1 millisecond min drift
		// for small TTLs.
		drift := int64(float64(ttl*1000)*r.clockDrift) + 2
		validityTime := int64(ttl*1000) - (time.Now().UnixNano()/1e6 - startTime) - drift

		if n >= r.quorum && validityTime > 0 {
			lockRet.State = true
			return
		}

		r.actAsync(func(pool Pool) bool {
			return r.unlock(pool, lockRet) == nil
		})

		delay := randInt64(r.retryDelay/2, r.retryDelay)
		time.Sleep(time.Duration(delay) * time.Millisecond)

		r.retryCount--
	}
}

// 释放锁
func (r *RedLock) Unlock(lockRet LockRet) bool {
	n := r.actAsync(func(pool Pool) bool {
		return r.unlock(pool, lockRet) == nil
	})
	return n >= r.quorum
}

func (r *RedLock) actAsync(actFn func(Pool) bool) int {
	ch := make(chan bool)
	for _, pool := range r.pools {
		go func(pool Pool) {
			ch <- actFn(pool)
		}(pool)
	}
	n := 0
	if <-ch {
		n++
	}
	return n
}

func (r *RedLock) lock(pool Pool, lockRet LockRet, ttl int) error {
	conn := pool.Get()
	defer conn.Close()

	status, err := redis.String(conn.Do("SET", lockRet.Resource, lockRet.Token, "NX", "PX", ttl*1000))
	if status == "" {
		err = errors.New("lock failed")
	}
	return err
}

func (r *RedLock) unlock(pool Pool, lockRet LockRet) error {
	conn := pool.Get()
	defer conn.Close()

	deleteScript := redis.NewScript(1, `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)
	status, err := redis.Int64(deleteScript.Do(conn, lockRet.Resource, lockRet.Token))
	if status == 0 {
		err = errors.New("unlock failed")
	}
	return err
}

func randInt64(min, max int64) int64 {
	rand.Seed(time.Now().UnixNano())
	randNum := rand.Int63n(max - min)
	randNum = randNum + min
	return randNum
}

func uniqId() string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	src := rand.NewSource(time.Now().UnixNano())
	n := 32
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}
