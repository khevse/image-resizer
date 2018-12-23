package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

type cacheItem struct {
	Key     string
	Data    []byte
	Expired time.Time
}

// Cache - internal cache
type Cache struct {
	maxFileSize    int64
	lifetime       time.Duration
	cleanerTimeout time.Duration
	payload        []*cacheItem

	closed int32
	mu     sync.Mutex
}

// New return new cache object
func New(maxFileSize int64, maxItems int, lifetime, cleanerTimeout time.Duration) *Cache {

	c := &Cache{
		maxFileSize:    maxFileSize,
		lifetime:       lifetime,
		cleanerTimeout: cleanerTimeout,
		payload:        make([]*cacheItem, 0, maxItems),
	}

	c.runAutoCleaner()
	return c
}

// Add new value to cache
func (c *Cache) Add(key string, data []byte) {

	if datalen := len(data); datalen == 0 || atomic.LoadInt64(&c.maxFileSize) < int64(datalen) {
		return // ignore file
	}

	lifetime := time.Duration(atomic.LoadInt64((*int64)(&c.lifetime)))

	newItem := &cacheItem{
		Key:     key,
		Data:    make([]byte, len(data)),
		Expired: time.Now().Add(lifetime),
	}

	copy(newItem.Data, data)

	c.mu.Lock()
	defer c.mu.Unlock()

	if (cap(c.payload) - len(c.payload)) > 0 {
		c.payload = append(c.payload, newItem)
	} else {
		lastIndex := len(c.payload) - 1
		c.payload[lastIndex] = newItem
	}
}

// Get return file if exist in cache
func (c *Cache) Get(key string) (data []byte, ok bool) {

	c.mu.Lock()
	defer c.mu.Unlock()

	for i, item := range c.payload {
		if item.Key != key {
			continue
		}

		data = make([]byte, len(item.Data))
		copy(data, item.Data)
		ok = true

		if i > 0 {
			// move item to start
			copy(c.payload[1:i+1], c.payload[:i])
			c.payload[0] = item
		}

		return data, ok
	}

	return nil, false
}

// Close cache (stop autoclean)
func (c *Cache) Close() error {
	atomic.StoreInt32(&c.closed, 1)
	return nil
}

func (c *Cache) runAutoCleaner() {

	timeout := time.Duration(atomic.LoadInt64((*int64)(&c.cleanerTimeout)))

	go func() {
		for {
			if atomic.LoadInt32(&c.closed) > 0 {
				return
			}

			time.Sleep(timeout)

			now := time.Now()

			func() {

				c.mu.Lock()
				defer c.mu.Unlock()

				var i int
				for i < len(c.payload) {
					item := c.payload[i]
					if now.Before(item.Expired) {
						i++
						continue
					}

					copy(c.payload[i:], c.payload[i+1:])
					c.payload = c.payload[:len(c.payload)-1]
				}
			}()
		}
	}()
}
