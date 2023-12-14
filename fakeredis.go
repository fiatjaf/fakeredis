package fakeredis

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/maps"
)

type FakeRedis struct {
	sync.Mutex
	values        map[string]string
	valueSlices   map[string][]string
	valueHashmaps map[string]map[string]string
	expirations   map[string]time.Time
	ctx           context.Context
	cancel        context.CancelFunc
}

func New() *FakeRedis {
	ctx, cancel := context.WithCancel(context.Background())

	f := FakeRedis{
		values:      make(map[string]string),
		valueSlices: make(map[string][]string),
		expirations: make(map[string]time.Time),
		ctx:         ctx,
		cancel:      cancel,
	}

	// delete expired keys
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Minute):
				f.cleanupExpired()
			}
		}
	}()

	return &f
}

func (f *FakeRedis) Close() {
	f.cancel()
}

func (f *FakeRedis) Ping() *redis.StatusCmd {
	cmd := redis.NewStatusCmd(f.ctx)
	cmd.SetVal("PONG")
	return cmd
}

func (f *FakeRedis) Get(key string) *redis.StringCmd {
	f.Lock()
	defer f.Unlock()
	cmd := redis.NewStringCmd(f.ctx)
	cmd.SetVal(f.getCheckExpiration(key))
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
	v, err := strconv.ParseInt(f.getCheckExpiration(key), 10, 64)
	if err != nil {
		cmd.SetErr(fmt.Errorf("value is not an integer or out of range: %w", err))
	} else {
		v++
		f.values[key] = strconv.FormatInt(v, 10)
		cmd.SetVal(v)
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

func (f *FakeRedis) HSet(key string, values ...any) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()

	cmd := redis.NewIntCmd(f.ctx)
	hashmap := f.hashmapGetCheckExpiration(key)
	if len(values) == 1 {
		switch agg := values[0].(type) {
		case []any:
			values = agg
		case map[string]string:
			values = make([]any, len(agg)*2)
			i := 0
			for k, v := range agg {
				values[i] = k
				i++
				values[i] = v
				i++
			}
		}
	} else if len(values)%2 != 0 {
		cmd.SetErr(fmt.Errorf("invalid number of arguments: %d", len(values)))
		return cmd
	}

	for i := 0; i < len(values); i += 2 {
		hashmap[fmt.Sprintf("%v", values[i])] = fmt.Sprintf("%v", values[i+1])
	}
	return cmd
}

func (f *FakeRedis) HGet(key string, field string) *redis.StringCmd {
	f.Lock()
	defer f.Unlock()
	hashmap := f.hashmapGetCheckExpiration(key)
	v, _ := hashmap[field]
	cmd := redis.NewStringCmd(f.ctx)
	cmd.SetVal(v)
	return cmd
}

func (f *FakeRedis) HDel(key string, fields ...string) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()
	var d int64
	hashmap := f.hashmapGetCheckExpiration(key)
	for _, field := range fields {
		_, exists := hashmap[field]
		if exists {
			delete(hashmap, field)
			d++
		}
	}
	if d > 0 {
		if len(hashmap) > 0 {
			f.valueHashmaps[key] = hashmap
		} else {
			delete(f.valueHashmaps, key)
		}
	}
	cmd := redis.NewIntCmd(f.ctx)
	cmd.SetVal(d)
	return cmd
}

func (f *FakeRedis) HGetAll(key string) *redis.MapStringStringCmd {
	f.Lock()
	defer f.Unlock()
	cmd := redis.NewMapStringStringCmd(f.ctx)
	cmd.SetVal(maps.Clone(f.hashmapGetCheckExpiration(key)))
	return cmd
}

func (f *FakeRedis) LPush(key string, values ...any) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()
	newValues := []string{}
	for _, v := range values {
		newValues = append(newValues, fmt.Sprintf("%v", v))
	}
	f.valueSlices[key] = append(newValues, f.valueSlices[key]...)
	cmd := redis.NewIntCmd(f.ctx)
	cmd.SetVal(int64(len(key)))
	return cmd
}

func (f *FakeRedis) RPop(key string) *redis.StringCmd {
	f.Lock()
	defer f.Unlock()

	cmd := redis.NewStringCmd(f.ctx)
	l := len(f.sliceGetCheckExpiration(key))
	if l == 0 {
		cmd.SetVal("")
		return cmd
	}
	v := f.valueSlices[key][l-1]
	f.valueSlices[key] = f.valueSlices[key][:l-1]
	cmd.SetVal(v)
	return cmd
}

// internal methods
func (f *FakeRedis) getCheckExpiration(key string) string {
	v := f.values[key]
	exp, ok := f.expirations[key]
	if ok && exp.Before(time.Now()) {
		v = ""
		delete(f.values, key)
		delete(f.expirations, key)
	}
	return v
}

func (f *FakeRedis) sliceGetCheckExpiration(key string) []string {
	v, ok := f.valueSlices[key]
	if !ok {
		return v
	}
	exp, ok := f.expirations[key]
	if ok && exp.Before(time.Now()) {
		v = nil
		delete(f.valueSlices, key)
		delete(f.expirations, key)
	}
	return v
}

func (f *FakeRedis) hashmapGetCheckExpiration(key string) map[string]string {
	var hashmap map[string]string
	if exp, ok := f.expirations[key]; !ok || !exp.Before(time.Now()) {
		hashmap, _ = f.valueHashmaps[key]
	}
	if hashmap == nil {
		hashmap = make(map[string]string)
	}
	return hashmap
}

func (f *FakeRedis) cleanupExpired() {
	f.Lock()
	defer f.Unlock()
	for k, v := range f.expirations {
		if v.Before(time.Now()) {
			f.del(k)
		}
	}
}

func (f *FakeRedis) del(key string) {
	delete(f.expirations, key)
	delete(f.values, key)
	delete(f.valueSlices, key)
	delete(f.valueHashmaps, key)
}
