package fakeredis

import "time"

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
