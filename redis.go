package redlock_go

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

type Pool interface {
	Get() redis.Conn
}

/*
["redis://[password]@[host]:[port]", ...]
*/
func newPools(servers []string) []Pool {
	var pools []Pool
	for _, server := range servers {
		pool := newPool(server)
		pools = append(pools, pool)
	}
	return pools
}

func newPool(server string) *redis.Pool {
	return &redis.Pool{
		MaxActive:   40,
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Wait:        true,
		Dial: func() (conn redis.Conn, err error) {
			conn, err = redis.DialURL(server)
			if err != nil {
				panic(err)
			}
			return
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := conn.Do("PING")
			return err
		},
	}
}
