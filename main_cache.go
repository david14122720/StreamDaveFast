package main

import (
	"container/list"
	"hash/fnv"
	"sync"
	"time"
)

const numShards = 16

type cacheEntry struct {
	key        string
	data       []byte
	size       int
	lastAccess time.Time
}

type shard struct {
	mu      sync.Mutex
	entries map[string]*list.Element
	lru     *list.List
	bytes   int64
}

type SegmentCache struct {
	shards      [numShards]*shard
	maxEntry    int
	maxTotal    int64
	maxPerShard int64
}

func NewSegmentCache(maxEntry int, maxTotal int64) *SegmentCache {
	c := &SegmentCache{
		maxEntry:    maxEntry,
		maxTotal:    maxTotal,
		maxPerShard: maxTotal / int64(numShards),
	}
	for i := range c.shards {
		c.shards[i] = &shard{
			entries: make(map[string]*list.Element),
			lru:     list.New(),
		}
	}
	return c
}

func (c *SegmentCache) shardFor(key string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return c.shards[h.Sum32()%numShards]
}

func (c *SegmentCache) Get(key string) ([]byte, bool) {
	s := c.shardFor(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	el, ok := s.entries[key]
	if !ok {
		return nil, false
	}
	e := el.Value.(*cacheEntry)
	e.lastAccess = time.Now()
	s.lru.MoveToFront(el)
	return e.data, true
}

func (c *SegmentCache) Put(key string, data []byte) {
	size := len(data)
	if size > c.maxEntry {
		return
	}
	s := c.shardFor(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	if el, ok := s.entries[key]; ok {
		e := el.Value.(*cacheEntry)
		s.bytes -= int64(e.size)
		e.data = data
		e.size = size
		e.lastAccess = time.Now()
		s.bytes += int64(size)
		s.lru.MoveToFront(el)
		return
	}

	for s.bytes+int64(size) > c.maxPerShard && s.lru.Len() > 0 {
		oldest := s.lru.Back()
		if oldest == nil {
			break
		}
		s.lru.Remove(oldest)
		e := oldest.Value.(*cacheEntry)
		delete(s.entries, e.key)
		s.bytes -= int64(e.size)
	}

	e := &cacheEntry{key: key, data: data, size: size, lastAccess: time.Now()}
	el := s.lru.PushFront(e)
	s.entries[key] = el
	s.bytes += int64(size)
}

func (c *SegmentCache) Delete(key string) {
	s := c.shardFor(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	if el, ok := s.entries[key]; ok {
		e := el.Value.(*cacheEntry)
		s.lru.Remove(el)
		delete(s.entries, key)
		s.bytes -= int64(e.size)
	}
}

func (c *SegmentCache) Len() int {
	n := 0
	for _, s := range c.shards {
		s.mu.Lock()
		n += s.lru.Len()
		s.mu.Unlock()
	}
	return n
}

func (c *SegmentCache) Bytes() int64 {
	var n int64
	for _, s := range c.shards {
		s.mu.Lock()
		n += s.bytes
		s.mu.Unlock()
	}
	return n
}

func (c *SegmentCache) CleanOlderThan(age time.Duration) int {
	cutoff := time.Now().Add(-age)
	removed := 0
	for _, s := range c.shards {
		s.mu.Lock()
		for el := s.lru.Back(); el != nil; {
			next := el.Prev()
			e := el.Value.(*cacheEntry)
			if e.lastAccess.After(cutoff) {
				break
			}
			s.lru.Remove(el)
			delete(s.entries, e.key)
			s.bytes -= int64(e.size)
			removed++
			el = next
		}
		s.mu.Unlock()
	}
	return removed
}
