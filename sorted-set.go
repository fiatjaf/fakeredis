package fakeredis

import (
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slices"
)

func zScoreCompare(a, b redis.Z) int {
	if a.Score > b.Score {
		return 1
	} else if a.Score < b.Score {
		return -1
	}
	return 0
}

func (f *FakeRedis) ZAdd(key string, values ...redis.Z) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()
	return f.zadd(key, false, values...)
}

func (f *FakeRedis) ZAddNX(key string, values ...redis.Z) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()
	return f.zadd(key, true, values...)
}

func (f *FakeRedis) zadd(key string, nx bool, values ...redis.Z) *redis.IntCmd {
	cmd := redis.NewIntCmd(f.ctx)
	zset, ok := f.sortedSetGetCheckExpiration(key)
	if !ok {
		zset = make([]redis.Z, 0, len(values))
	}

	var n int64
	for _, v := range values {
		idx := slices.IndexFunc(zset, func(z redis.Z) bool { return z.Member == v.Member })
		if idx != -1 {
			// member exists -- if not nx
			if nx {
				continue
			}
			// just change the score
			curr := zset[idx].Score
			switch {
			case v.Score == curr:
				// same score, do nothing
			case v.Score < curr:
				// new score is lower, so we must move it towards the beginning
				next, _ := slices.BinarySearchFunc(zset[:idx], v, zScoreCompare)
				copy(zset[next+1:idx+1], zset[next:idx])
				zset[next] = v
			case v.Score > curr:
				// new score is greater, so we must move it towards the end
				next, _ := slices.BinarySearchFunc(zset[idx:], v, zScoreCompare)
				copy(zset[idx:idx+next], zset[idx+1:idx+next])
				zset[idx+next-1] = v
			}
		} else {
			// member doesn't exist, add it to the correct sorted position
			n++
			idx, _ := slices.BinarySearchFunc(zset, v, zScoreCompare)
			zset = append(zset, redis.Z{}) // bogus
			copy(zset[idx+1:], zset[idx:])
			zset[idx] = v
		}
	}
	f.valueSortedSets[key] = zset
	cmd.SetVal(n)
	return cmd
}

func (f *FakeRedis) ZCard(key string) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()
	zset, _ := f.sortedSetGetCheckExpiration(key)
	cmd := redis.NewIntCmd(f.ctx)
	cmd.SetVal(int64(len(zset)))
	return cmd
}

func (f *FakeRedis) ZRem(key string, members ...string) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()
	cmd := redis.NewIntCmd(f.ctx)
	zset, ok := f.sortedSetGetCheckExpiration(key)
	if !ok {
		return cmd
	}

	var n int64
	for _, member := range members {
		idx := slices.IndexFunc(zset, func(z redis.Z) bool { return z.Member == member })
		if idx != -1 {
			// exists, so remove it
			n++
			copy(zset[idx:], zset[idx+1:])
			zset = zset[0 : len(zset)-1]
		}
	}

	f.valueSortedSets[key] = zset
	cmd.SetVal(n)
	return cmd
}

func (f *FakeRedis) ZRangeByScore(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	f.Lock()
	defer f.Unlock()
	cmd := redis.NewStringSliceCmd(f.ctx)
	zset, ok := f.sortedSetGetCheckExpiration(key)
	if !ok {
		return cmd
	}

	maxMembers := len(zset)
	var _opt redis.ZRangeBy
	if opt != nil {
		_opt = *opt
	}

	if _opt.Offset != 0 {
		maxMembers = maxMembers - int(_opt.Offset)
		zset = zset[_opt.Offset:]
	}

	minV := parseInt(_opt.Min)
	minIdx := 0
	if minV != 0 {
		minIdx, _ = slices.BinarySearchFunc(zset, redis.Z{Score: float64(minV)}, zScoreCompare)
	}
	maxV := parseInt(_opt.Max)
	maxIdx := len(zset)
	if maxV != 0 {
		maxIdx, _ = slices.BinarySearchFunc(zset, redis.Z{Score: float64(maxV)}, zScoreCompare)
	}
	if _opt.Count != 0 {
		maxIdx = min(maxMembers, minIdx+int(_opt.Count))
		maxMembers = int(_opt.Count)
	}

	values := make([]string, 0, maxMembers)
	for _, v := range zset[minIdx:maxIdx] {
		values = append(values, v.Member.(string))
	}

	cmd.SetVal(values)
	return cmd
}

func (f *FakeRedis) ZScore(key string, member string) *redis.IntCmd {
	f.Lock()
	defer f.Unlock()
	cmd := redis.NewIntCmd(f.ctx)
	zset, ok := f.sortedSetGetCheckExpiration(key)
	if !ok {
		return cmd
	}
	idx := slices.IndexFunc(zset, func(z redis.Z) bool { return z.Member == member })
	if idx != -1 {
		cmd.SetVal(int64(zset[idx].Score))
	}
	return cmd
}

func (f *FakeRedis) sortedSetGetCheckExpiration(key string) ([]redis.Z, bool) {
	v, exists := f.valueSortedSets[key]
	if !exists {
		return nil, false
	}
	exp, ok := f.expirations[key]
	if ok && exp.Before(time.Now()) {
		delete(f.valueSortedSets, key)
		delete(f.expirations, key)
		return nil, false
	}
	return v, true
}
