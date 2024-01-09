package fakeredis

import (
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func (f *FakeRedis) Get(key string) *redis.StringCmd {
	f.Lock()
	defer f.Unlock()
	cmd := redis.NewStringCmd(f.ctx)
	v, ok := f.getCheckExpiration(key)
	if !ok {
		cmd.SetErr(fmt.Errorf("not found"))
	} else {
		cmd.SetVal(v)
	}
	return cmd
}

func (f *FakeRedis) Set(key string, value any, expiration time.Duration) *redis.StatusCmd {
	f.Lock()
	defer f.Unlock()
	f.values[key] = fmt.Sprintf("%v", value)
	if expiration > 0 {
		f.expirations[key] = time.Now().Add(expiration)
	} else {
		delete(f.expirations, key)
	}
	cmd := redis.NewStatusCmd(f.ctx)
	cmd.SetVal("OK")
	return cmd
}

func (f *FakeRedis) Incr(key string) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()

	cmd := redis.NewIntCmd(f.ctx)
	v, ok := f.getCheckExpiration(key)
	if !ok {
		cmd.SetErr(fmt.Errorf("not found"))
	} else {
		d, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			cmd.SetErr(fmt.Errorf("value is not an integer or out of range: %w", err))
		} else {
			d++
			f.values[key] = strconv.FormatInt(d, 10)
			cmd.SetVal(d)
		}
	}
	return cmd
}

func (f *FakeRedis) SetNX(key string, value any, expiration time.Duration) *redis.BoolCmd {
	f.Lock()
	defer f.Unlock()
	cmd := redis.NewBoolCmd(f.ctx)
	_, ok := f.values[key]
	if ok {
		exp, ok2 := f.expirations[key]
		if !ok2 || exp.After(time.Now()) {
			cmd.SetVal(false)
			return cmd
		}
	}
	f.values[key] = fmt.Sprintf("%v", value)
	if expiration > 0 {
		f.expirations[key] = time.Now().Add(expiration)
	} else {
		delete(f.expirations, key)
	}
	cmd.SetVal(true)
	return cmd
}

func (f *FakeRedis) getCheckExpiration(key string) (string, bool) {
	v, exists := f.values[key]
	if !exists {
		return "", false
	}
	exp, ok := f.expirations[key]
	if ok && exp.Before(time.Now()) {
		delete(f.values, key)
		delete(f.expirations, key)
		return "", false
	}
	return v, true
}
