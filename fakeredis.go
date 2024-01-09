package fakeredis

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type FakeRedis struct {
	sync.Mutex
	values          map[string]string
	valueSlices     map[string][]string
	valueHashmaps   map[string]map[string]string
	valueSortedSets map[string][]redis.Z
	expirations     map[string]time.Time
	ctx             context.Context
	cancel          context.CancelFunc
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
