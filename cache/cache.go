package cache

import (
	"log"
	"sync"
)

type CacheI interface {
	Get(key any) (data any, ok bool)
	Set(key any, value any)
	Delete(key any)
	Destroy()
	GetAll() (newmap map[any]any)
	SubCache(key any) CacheI
	Locker() *sync.Mutex
	RWLocker() *sync.RWMutex
}

// 使用sync.Map实现CacheI
type syncCache struct {
	dataMap sync.Map
}

// 获取值
func (c *syncCache) Get(key any) (data any, ok bool) {
	data, ok = c.dataMap.Load(key)
	return
}

// 设置值
func (c *syncCache) Set(key any, value any) {
	c.dataMap.Store(key, value)
}

// 删除值
func (c *syncCache) Delete(key any) {
	c.dataMap.Delete(key)
}

// 清除自身所有kv 以及后续所有子节点
func (c *syncCache) Destroy() {
	c.dataMap.Range(func(key, value any) bool {
		if subcache, ok := value.(*syncCache); ok {
			log.Println("enter subcache:", key)
			subcache.Destroy()
			log.Println("out subcache and clear:", key)
		}
		c.dataMap.Delete(key)
		return true
	})
	log.Println("all cache clear")
}

// 获取所有的键值对
func (c *syncCache) GetAll() (newmap map[any]any) {
	newmap = make(map[any]any)
	c.dataMap.Range(func(key, value any) bool {
		newmap[key] = value
		return true
	})
	return
}

// 树状缓存
func (c *syncCache) SubCache(key any) CacheI {
	subcache, ok := c.Get(key)
	if ok {
		ret, ok := subcache.(CacheI)
		if ok {
			return ret
		}
		log.Println("cache broke")
	}
	ret := SyncCacheInit()
	c.Set(key, ret)
	log.Printf("initialize the subcache [%s]", key)
	return ret
}
func (s *syncCache) Locker() (mutex *sync.Mutex) {
	rawMu, ok := s.Get("mutex")
	if ok {
		mutex = rawMu.(*sync.Mutex)
	} else {
		var mu sync.Mutex
		s.Set("mutex", &mu)
		mutex = &mu
	}
	return
}
func (s *syncCache) RWLocker() (mutex *sync.RWMutex) {
	rawMu, ok := s.Get("rwmutex")
	if ok {
		mutex = rawMu.(*sync.RWMutex)
	} else {
		var mu sync.RWMutex
		s.Set("mutex", &mu)
		mutex = &mu
	}
	return
}

// 实例化
func SyncCacheInit() (ret CacheI) {
	ret = &syncCache{}
	return
}
