package memgroupcache

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/groupcache"
	"github.com/stretchr/testify/assert"
)

func TestMemGroup(t *testing.T) {
	prefix := "sars"
	var n int64 = 0
	var numTimesGetCalled = 0
	now := func() int64 { return n }
	mg := NewMemGroup("groupname", 1<<20, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			numTimesGetCalled++
			return dest.SetString(prefix + key + fmt.Sprintf("%d", now()))
		}), MemGroupOptions{
		IgnoreAfter:  time.Nanosecond * 10,
		RefreshAfter: time.Nanosecond * 5,
	})
	mg.now = now
	sinkStr := ""
	err := mg.Get(nil, "key1", groupcache.StringSink(&sinkStr))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, numTimesGetCalled)
	assert.Equal(t, prefix+"key10", sinkStr)

	sinkStr = ""
	err = mg.Get(nil, "key1", groupcache.StringSink(&sinkStr))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, numTimesGetCalled)
	assert.Equal(t, prefix+"key10", sinkStr)

	n = 5
	err = mg.Get(nil, "key1", groupcache.StringSink(&sinkStr))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, numTimesGetCalled)
	assert.Equal(t, prefix+"key15", sinkStr)

	for ; n < 10; n++ {
		f := func() {
			sinkStr = ""
			err = mg.Get(nil, "key1", groupcache.StringSink(&sinkStr))
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 2, numTimesGetCalled)
			assert.Equal(t, prefix+"key15", sinkStr)
		}
		f()
		go f()
	}
}
