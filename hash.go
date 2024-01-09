package fakeredis

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/maps"
)

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
