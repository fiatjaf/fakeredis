package fakeredis

import (
	"time"

	"github.com/redis/go-redis/v9"
)

func (f *FakeRedis) Ping() *redis.StatusCmd {
	cmd := redis.NewStatusCmd(f.ctx)
	cmd.SetVal("PONG")
	return cmd
}

func (f *FakeRedis) Del(keys ...string) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()
	for _, key := range keys {
		f.del(key)
	}
	cmd := redis.NewIntCmd(f.ctx)
	cmd.SetVal(int64(len(keys)))
	return cmd
}

func (f *FakeRedis) Expire(key string, expiration time.Duration) *redis.StatusCmd {
	f.Lock()
	defer f.Unlock()
	f.expirations[key] = time.Now().Add(expiration)
	cmd := redis.NewStatusCmd(f.ctx)
	cmd.SetVal("OK")
	return cmd
}
