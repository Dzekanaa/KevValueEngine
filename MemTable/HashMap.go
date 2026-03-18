package memtable

import (
	"bytes"
)

const ArraySize = MemtableSize

//		func (reciever) nameOfFunction (paramaters) returnValue {}
//			 ^ like the this keyword
//				used only in methods
//				when used, function is called like:	this.nameOfFunction

// HashMap structure
type HashMap struct {
	array [ArraySize]*bucketElem
	size  int
}

// bucket structure
type bucketElem struct {
	entry Entry
	next  *bucketElem
}

// bucketNode structure

// Insert
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

// Get
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

// Delete
func (h *HashMap) Delete(key []byte) bool {
	index := hash(key)
	curr := h.array[index]

	for curr != nil {
		if bytes.Equal(curr.entry.Key, key) {
			curr.entry.Tombstone = true
			curr.entry.Value = nil
			return true
		}
		curr = curr.next
	}

	// previously, physical deletion
	//
	// if h.array[index] != nil && bytes.Equal(h.array[index].entry.Key, key) {
	// 	h.array[index] = h.array[index].next
	// 	h.size--
	// 	return true
	// }

	// prev := h.array[index]
	// for prev != nil && prev.next != nil {
	// 	if bytes.Equal(prev.next.entry.Key, key) {
	// 		prev.next = prev.next.next
	// 		h.size--
	// 		return true
	// 	}
	// 	prev = prev.next
	// }

	return false
}

// hash
func hash(key []byte) int {
	sum := 0
	for _, b := range key {
		sum += int(b)
	}
	return sum % ArraySize
}

// Init
func NewHashMap() *HashMap {
	return &HashMap{}
}
