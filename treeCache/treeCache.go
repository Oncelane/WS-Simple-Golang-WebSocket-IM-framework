package treeCache

import (
	"sync"
)

type CacheI interface {
	Get(key any) (data any, ok bool)
	Set(key any, value any)
	Delete(key any)
	Destroy()
	GetAll() (newmap map[any]any)
	SubCache(key string) CacheI
	Locker() *sync.RWMutex
}

// sync.Map implement CacheI
type syncCache struct {
	dataMap sync.Map
	rw      sync.RWMutex
	pre     *syncCache
	nexts   map[string]*syncCache
	name    string
}

// get by key
func (c *syncCache) Get(key any) (data any, ok bool) {
	data, ok = c.dataMap.Load(key)
	return
}

// set k-v
func (c *syncCache) Set(key any, value any) {
	c.dataMap.Store(key, value)
}

// delete by key
func (c *syncCache) Delete(key any) {
	c.dataMap.Delete(key)
}

// delete all k-v
func (c *syncCache) Clear() {
	c.dataMap.Range(func(key, value any) bool {
		c.Delete(key)
		return true
	})
}

// delete this node and its children node
func (c *syncCache) Destroy() {
	// delete subcache
	for _, subcache := range c.nexts {
		subcache.Destroy()
	}
	clear(c.nexts)
	// delete all key value
	c.Clear()
	if c.pre != nil {
		c.pre.Delete(c.name)
	}
	// log.Println("all cache clear")
}

// get all k-v
func (c *syncCache) GetAll() (newmap map[any]any) {
	newmap = make(map[any]any)
	c.dataMap.Range(func(key, value any) bool {
		newmap[key] = value
		return true
	})
	return
}

// create/get children note
func (c *syncCache) SubCache(key string) CacheI {
	subcache, ok := c.nexts[key]
	if ok {
		return subcache
	}
	subcache = NewSyncCache()
	c.nexts[key] = subcache
	subcache.pre = c
	subcache.name = key
	return subcache
}

// rwlock
func (c *syncCache) Locker() *sync.RWMutex {
	return &c.rw
}

func NewSyncCache() (ret *syncCache) {
	ret = &syncCache{
		name:  "root",
		pre:   nil,
		nexts: map[string]*syncCache{},
	}
	return
}
