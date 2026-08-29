// Package memtable implements an in-memory data structure for holding
// the most recently written key-value pairs before they are flushed to disk
package memtable

import (
	"fmt"
	"sync"
)

// Backup default value for the max size of a Memtable
// Ensures the explicitly entered value is valid
const MaxSize = 1000

// Entry represents a single key-value pair stored in the memtable.
// Tombstone indicates this is a logical delete (key marked for deletion).
type Entry struct {
	Value     []byte
	Tombstone bool
}

// Memtable is an in-memory hash map that stores key-value pairs before
// they are flushed to SSTable. It tracks insertion order and provides
// thread-safe Put, Get, and Delete operations.
type Memtable struct {
	mu      sync.RWMutex
	data    map[string]*Entry
	maxSize int // max number of entries before flush is needed
	isFull  bool
}

// New creates a new Memtable with a maximum capacity of maxSize entries.
func New(maxSize int) *Memtable {
	if maxSize <= 0 {
		maxSize = MaxSize // backup default
	}
	return &Memtable{
		data:    make(map[string]*Entry),
		maxSize: maxSize,
		isFull:  false,
	}
}

// Put adds or updates a key-value pair in the memtable.
// Returns an error if the memtable is already full.
func (m *Memtable) Put(key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if key already exists (update, not insert)
	_, exists := m.data[key]
	if !exists && len(m.data) >= m.maxSize {
		m.isFull = true
		return fmt.Errorf("memtable is full: %d / %d entries", len(m.data), m.maxSize)
	}

	m.data[key] = &Entry{
		Value:     value,
		Tombstone: false,
	}

	return nil
}

// Get retrieves the value associated with the given key.
// Returns (value, found, tombstone).
// If found is false, the key does not exist in the memtable.
// If tombstone is true, the key was deleted but the marker is still there.
func (m *Memtable) Get(key string) ([]byte, bool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, found := m.data[key]
	if !found {
		return nil, false, false
	}

	return entry.Value, true, entry.Tombstone
}

// Delete marks a key as deleted (logical delete via tombstone marker).
// Returns an error if the memtable is already full.
func (m *Memtable) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we have space for the tombstone marker
	_, exists := m.data[key]
	if !exists && len(m.data) >= m.maxSize {
		m.isFull = true
		return fmt.Errorf("memtable is full: %d / %d entries", len(m.data), m.maxSize)
	}

	m.data[key] = &Entry{
		Value:     nil,
		Tombstone: true,
	}

	return nil
}

// IsFull returns true if the memtable has reached its maximum capacity.
func (m *Memtable) IsFull() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isFull
}

// Size returns the current number of entries in the memtable.
func (m *Memtable) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}
