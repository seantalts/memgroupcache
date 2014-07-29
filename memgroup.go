package memgroupcache

import (
	"encoding/json"
	"time"

	"github.com/golang/groupcache"
)

type MemGroup struct {
	*groupcache.Group
	options MemGroupOptions
	getter  groupcache.Getter
	now     func() int64
}

type MemGroupOptions struct {
	RefreshAfter time.Duration

	// IgnoreAfter also serves as a chunking mechanism for the keys: clients add
	// the current time divided by IgnoreAfter to the key when looking in
	// groupcache.
	IgnoreAfter time.Duration
}

type memGroupKey struct {
	OriginalKey string
	Version     int64
}

func makeGetter(getter groupcache.Getter) groupcache.Getter {
	// Called when the cache misses.
	return groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			mkey := memGroupKey{}
			err := json.Unmarshal([]byte(key), &mkey)
			if err != nil {
				panic(err)
			}
			return getter.Get(ctx, mkey.OriginalKey, dest)
		})
}

func NewMemGroup(name string, cacheBytes int64, getter groupcache.Getter,
	options MemGroupOptions) MemGroup {
	group := groupcache.NewGroup(name, cacheBytes, makeGetter(getter))

	return MemGroup{
		Group:   group,
		options: options,
		getter:  getter,
		now:     func() int64 { return time.Now().UnixNano() },
	}
}

// Not implemented yet - not sure why GetGroup exists yet.
func GetMemGroup(name string) MemGroup {
	panic("Not implemented")
}

// Called before talking to the underlying groupcache.
func (m MemGroup) Get(ctx groupcache.Context, key string,
	dest groupcache.Sink) (err error) {

	getVersion := func(d time.Duration) (chan string, func()) {
		c := make(chan string)
		return c, func() {
			sinkStr := ""
			var mkey []byte
			mkey, err = json.Marshal(memGroupKey{
				OriginalKey: key,
				Version:     m.now() / int64(d),
			})
			if err != nil {
				panic(err)
			}
			err = m.Group.Get(ctx, string(mkey), groupcache.StringSink(&sinkStr))
			c <- sinkStr
		}
	}

	refreshChan, refreshFunc := getVersion(m.options.RefreshAfter)
	go refreshFunc()
	ignoreChan, ignoreFunc := getVersion(m.options.IgnoreAfter)
	go ignoreFunc()
	select {
	case s := <-refreshChan:
		dest.SetString(s)
	case s := <-ignoreChan:
		dest.SetString(s)
	}
	return err
}
