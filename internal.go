package fakeredis

import "time"

// internal methods
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
