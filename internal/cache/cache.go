// Package cache implements an in-memory LRU (Least Recently Used) cache,
// used by the Block Manager to avoid re-reading blocks from disk.
package cache

import (
	"container/list"
)

// CacheEntry holds the data stored in a single list element.
type CacheEntry struct {
	key   string
	value []byte
}

// LRUCache is a fixed-capacity cache that evicts the least recently
// used entry once it is full. It combines a doubly linked list, which
// tracks usage order, with a hash map for O(1) lookups.
type LRUCache struct {
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

// NewLRUCache creates an LRUCache that holds at most capacity entries.
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

// Get retrieves the value stored under key. If found, the entry is
// moved to the front of the list, marking it as most recently used.
func (c *LRUCache) Get(key string) ([]byte, bool) {
	element, found := c.items[key]
	if !found {
		return nil, false
	}

	c.order.MoveToFront(element)
	entry := element.Value.(*CacheEntry)
	return entry.value, true
}

// Put adds or updates the value stored under key. If the cache is at
// capacity and key is not already present, the least recently used
// entry is evicted first.
func (c *LRUCache) Put(key string, value []byte) {
	if element, found := c.items[key]; found {
		c.order.MoveToFront(element)
		entry := element.Value.(*CacheEntry)
		entry.value = value
		return
	}

	if c.order.Len() >= c.capacity {
		c.evict()
	}

	newEntry := &CacheEntry{key: key, value: value}
	newElement := c.order.PushFront(newEntry)
	c.items[key] = newElement
}

// evict removes the least recently used entry, i.e. the one at the
// back of the list.
func (c *LRUCache) evict() {
	lastElement := c.order.Back()
	if lastElement != nil {
		entry := lastElement.Value.(*CacheEntry)
		delete(c.items, entry.key)
		c.order.Remove(lastElement)
	}
}

// Delete removes the entry stored under key, if present.
func (c *LRUCache) Delete(key string) {
	if element, found := c.items[key]; found {
		c.order.Remove(element)
		delete(c.items, key)
	}
}
