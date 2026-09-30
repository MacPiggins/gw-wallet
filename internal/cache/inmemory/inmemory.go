package inmemory

import (
	"sync"
	"time"
)

type item struct {
	value      any
	expiration int64 // Unix-время в наносекундах
}

type Cache struct {
	data map[string]*item
	mu   sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{data: make(map[string]*item)}
}

// Set кладёт значение с TTL (duration > 0) или без истечения
func (c *Cache) Set(key string, value any, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp int64
	if duration > 0 {
		exp = time.Now().Add(duration).UnixNano()
	}
	c.data[key] = &item{value: value, expiration: exp}
}

// Get возвращает значение и флаг наличия; если истёк — считается отсутствующим
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	it, ok := c.data[key]
	if !ok {
		return nil, false
	}
	if it.expiration > 0 && time.Now().UnixNano() > it.expiration {
		// Можно удалить сразу при чтении (lazy delete)
		delete(c.data, key)
		return nil, false
	}
	return it.value, true
}

// Delete удаляет ключ
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}
