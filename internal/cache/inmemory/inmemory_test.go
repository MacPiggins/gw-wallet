package inmemory

import (
	"sync"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache()

	key := "test-key"
	value := "test-value"

	c.Set(key, value, 0) // без TTL

	got, ok := c.Get(key)
	if !ok {
		t.Fatalf("expected key %q to exist, but not found", key)
	}
	if got != value {
		t.Errorf("expected value %q, got %q", value, got)
	}
}

func TestCache_GetExpired(t *testing.T) {
	c := NewCache()
	key := "expired-key"
	val := "some-value"
	ttl := 1 * time.Millisecond

	c.Set(key, val, ttl)

	// Ждём, пока ключ точно истечёт
	time.Sleep(2 * time.Millisecond)

	_, ok := c.Get(key)
	if ok {
		t.Error("expected expired key to be missing, but it was found")
	}
}

func TestCache_Delete(t *testing.T) {
	c := NewCache()
	key := "del-key"

	c.Set(key, "value", 0)

	c.Delete(key)

	_, ok := c.Get(key)
	if ok {
		t.Error("expected deleted key to be missing")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := NewCache()
	var wg sync.WaitGroup
	concurrency := 20
	opsPerGoroutine := 500

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := "key-" + string(rune(id)) + "-" + string(rune(j))
				c.Set(key, id*100+j, 10*time.Second)
				_, _ = c.Get(key) // просто проверяем, что не падает
			}
		}(i)
	}

	wg.Wait()
}

func TestCache_TTLBehaviorWithLazyDelete(t *testing.T) {
	c := NewCache()
	key := "lazy-key"
	val := 42
	ttl := 2 * time.Millisecond

	c.Set(key, val, ttl)

	time.Sleep(3 * time.Millisecond)

	// При чтении происходит «lazy delete» — ключ удаляется, если истёк
	_, ok := c.Get(key)
	if ok {
		t.Error("expected expired key to be removed after Get (lazy delete)")
	}

	// Убедимся, что ключ действительно не находится
	_, ok = c.Get(key)
	if ok {
		t.Error("key should not be present after expired Get")
	}
}
