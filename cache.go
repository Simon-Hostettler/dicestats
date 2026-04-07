package dicestats

import "sync"

type Cache struct {
	mu sync.RWMutex
	m  map[string]*Distribution
}

func NewCache() *Cache {
	return &Cache{m: map[string]*Distribution{}}
}

func (c *Cache) Get(key string) (*Distribution, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	d, ok := c.m[key]
	if !ok {
		return nil, false
	}
	return d, true
}

func (c *Cache) Put(key string, d *Distribution) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = d
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = map[string]*Distribution{}
}
