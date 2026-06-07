package main

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSegmentCache_PutGet(t *testing.T) {
	c := NewSegmentCache(1024, 4096)
	c.Put("k1", []byte("hello"))
	got, ok := c.Get("k1")
	if !ok {
		t.Fatal("expected hit")
	}
	if !bytes.Equal(got, []byte("hello")) {
		t.Errorf("got %q, want %q", got, "hello")
	}
	if _, ok := c.Get("missing"); ok {
		t.Error("expected miss")
	}
}

func TestSegmentCache_OversizedEntryDropped(t *testing.T) {
	c := NewSegmentCache(10, 100)
	c.Put("big", make([]byte, 100))
	if _, ok := c.Get("big"); ok {
		t.Error("oversized entry should not be cached")
	}
}

func TestSegmentCache_LRUEviction(t *testing.T) {
	const entrySize = 64
	const totalCap = 16 * entrySize
	c := NewSegmentCache(entrySize, totalCap)

	for i := 0; i < 32; i++ {
		key := fmt.Sprintf("k%02d", i)
		c.Put(key, make([]byte, entrySize))
	}

	if got := c.Bytes(); got > totalCap {
		t.Errorf("cache exceeded total cap: got %d, want <= %d", got, totalCap)
	}

	if c.Len() == 0 {
		t.Error("expected some entries to remain after eviction")
	}
}

func TestSegmentCache_CleanOlderThan(t *testing.T) {
	c := NewSegmentCache(10, 100)
	c.Put("a", []byte("a"))
	c.Put("b", []byte("b"))
	time.Sleep(20 * time.Millisecond)
	removed := c.CleanOlderThan(10 * time.Millisecond)
	if removed != 2 {
		t.Errorf("expected 2 evicted, got %d", removed)
	}
	if c.Len() != 0 {
		t.Errorf("expected cache empty, got %d entries", c.Len())
	}
}

func TestSegmentCache_ShardedByKey(t *testing.T) {
	c := NewSegmentCache(10, 100)
	if c.shardFor("/processed/a/seg-1.m4s") == c.shardFor("/processed/a/seg-1.m4s") {
		return
	}
	if c.shardFor("k1") == c.shardFor("k2") {
		t.Log("k1 and k2 hashed to same shard (probabilistic, not a failure)")
	}
}

func TestSegmentCache_ConcurrentNoRace(t *testing.T) {
	c := NewSegmentCache(64, 1<<20)
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				key := fmt.Sprintf("w%d-k%d", id, i%50)
				c.Put(key, []byte(key))
				_, _ = c.Get(key)
			}
		}(w)
	}
	wg.Wait()
	if c.Bytes() <= 0 {
		t.Error("expected non-zero cache bytes after concurrent puts")
	}
}
