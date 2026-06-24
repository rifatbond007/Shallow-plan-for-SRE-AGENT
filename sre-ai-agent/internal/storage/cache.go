package storage

import (
	"container/list"
	"sync"
	"time"

	"github.com/rifatbond007/sre-ai-agent/pkg/metrics"
)

type item struct {
	key       string
	value     any
	expiresAt time.Time
}

type Cache struct {
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
	items    map[string]*list.Element
	lru      *list.List
}

func NewCache(maxSize int, ttl time.Duration) *Cache {
	return &Cache{
		maxSize: maxSize,
		ttl:     ttl,
		items:   make(map[string]*list.Element),
		lru:     list.New(),
	}
}

func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	elem, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		metrics.CacheMissesTotal.Inc()
		return nil, false
	}

	it := elem.Value.(*item)
	if time.Now().After(it.expiresAt) {
		metrics.CacheMissesTotal.Inc()
		c.mu.Lock()
		c.removeElement(elem)
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	c.lru.MoveToFront(elem)
	c.mu.Unlock()
	return it.value, true
}

func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.lru.MoveToFront(elem)
		elem.Value.(*item).value = value
		elem.Value.(*item).expiresAt = time.Now().Add(c.ttl)
		return
	}

	if c.lru.Len() >= c.maxSize {
		c.removeOldest()
	}

	it := &item{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	elem := c.lru.PushFront(it)
	c.items[key] = elem
}

func (c *Cache) removeElement(elem *list.Element) {
	c.lru.Remove(elem)
	delete(c.items, elem.Value.(*item).key)
}

func (c *Cache) removeOldest() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}
