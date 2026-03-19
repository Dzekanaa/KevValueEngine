package memtable

import (
	"bytes"
)

// arraysize is equal to MemTableSize
// this might change to improve code efficiency
const ArraySize = MemtableSize

// HashMap
// array - list of elements held in buckets
// size - number of non-tombstone elements
type HashMap struct {
	array [ArraySize]*bucketElem
	size  int
}

// HashMap element held in bucket
// entry - MemTable entry
// next - pointer to next element ih bucket
type bucketElem struct {
	entry Entry
	next  *bucketElem
}

// NewHashMap - Default Constructor
// returns empty chain linked HashMap
func NewHashMap() *HashMap {
	return &HashMap{}
}

// Size
// returns number of non-tombstone elements
func (h *HashMap) Size() int {
	return h.size
}

// Hash - hashes key into index
// key - array of bytes
// returns an integer/index to HashMap
func hash(key []byte) int {
	sum := 0
	for _, b := range key {
		sum += int(b)
	}
	return sum % ArraySize
}

// Put adds an element to the HashMap
// entry - the MemTable element we're inserting/putting in
func (h *HashMap) Put(entry Entry) {
	index := hash(entry.Key)
	curr := h.array[index]

	for curr != nil {
		if bytes.Equal(curr.entry.Key, entry.Key) {
			curr.entry = entry
			return
		}
		curr = curr.next
	}

	newElem := &bucketElem{
		entry: entry,
		next:  h.array[index],
	}
	h.array[index] = newElem
	h.size++
}

// Get tries to find our entry by key
// key - array of bytes, Key variable of some Entry struct
// returns the MemTable entry or an empty entry, and a success/fail flag as bool
func (h *HashMap) Get(key []byte) (Entry, bool) {
	index := hash(key)
	curr := h.array[index]

	for curr != nil {
		if bytes.Equal(curr.entry.Key, key) {
			if curr.entry.Tombstone {
				return Entry{}, false
			}
			return curr.entry, true
		}
		curr = curr.next
	}

	return Entry{}, false
}

// Delete finds an element by key and deletes it if it exists
// key - array of bytes, Key variable of some Entry struct
// returns success/fail flag as bool
func (h *HashMap) Delete(key []byte) bool {
	index := hash(key)
	curr := h.array[index]

	for curr != nil {
		if bytes.Equal(curr.entry.Key, key) {
			if curr.entry.Tombstone {
				return false
			}
			curr.entry.Tombstone = true
			curr.entry.Value = nil
			h.size--
			return true
		}
		curr = curr.next
	}

	return false
}
