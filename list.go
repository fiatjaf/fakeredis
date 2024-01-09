package fakeredis

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

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
