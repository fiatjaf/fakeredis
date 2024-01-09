package fakeredis

import (
	"math"
	"strconv"
	"time"
)

// internal methods
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

func parseInt(v string) int64 {
	p, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		switch v {
		case "+inf":
			return math.MaxInt64
		case "-inf":
			return math.MinInt64
		}
	}
	return p
}
