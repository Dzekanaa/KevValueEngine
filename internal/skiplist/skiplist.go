// Package skiplist implements a probabilistic skip list, used as one of
// the backing structures for the Memtable (spec 1.2[DZ1]).
package skiplist

import (
	"math/rand"
)

// Node is a single skip list element. Deleted marks a logical delete
// (tombstone) — the node stays in the list until the Memtable is flushed,
// so a later SSTable scan still knows the key was removed.
type Node struct {
	Key     string
	Value   []byte
	Deleted bool
	next    []*Node
}

// SkipList is a probabilistic data structure offering expected O(log n)
// search, insert, and delete, used as a Memtable backing structure.
type SkipList struct {
	maxHeight int
	head      *Node
	height    int
	size      int
}

// NewNode creates a node with next pointers for levels 0..level.
func NewNode(key string, value []byte, level int) *Node {
	return &Node{
		Key:   key,
		Value: value,
		next:  make([]*Node, level+1),
	}
}

// NewSkipList creates an empty SkipList with the given maximum height
func NewSkipList(maxheight int) *SkipList {
	return &SkipList{
		maxHeight: maxheight,
		head:      NewNode("", nil, maxheight),
		height:    0,
		size:      0,
	}
}

// roll returns a random level for a new node, via repeated coin flips:
// each level beyond 0 has a 50% chance, capped at maxHeight.
func (s *SkipList) roll() int {
	level := 0
	// possible ret values from rand are 0 and 1
	// we stop shen we get a 0
	for rand.Int31n(2) == 1 && level < s.maxHeight {
		level++
	}
	return level
}

// Get returns the value stored under key, and whether it was found.
// A logically deleted key is treated as not found.
func (s *SkipList) Get(key string) ([]byte, bool) {
	current := s.head
	for i := s.height; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].Key < key {
			current = current.next[i]
		}
	}
	current = current.next[0]
	if current != nil && current.Key == key && !current.Deleted {
		return current.Value, true
	}
	return nil, false
}

// Insert adds a new key-value pair, updates an existing one, or revives
// a previously deleted key.
func (s *SkipList) Insert(key string, value []byte) {
	update := make([]*Node, s.maxHeight+1)
	current := s.head

	for i := s.height; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].Key < key {
			current = current.next[i]
		}
		update[i] = current
	}

	target := current.next[0]
	if target != nil && target.Key == key {
		if target.Deleted {
			s.size++
		}
		target.Value = value
		target.Deleted = false
		return
	}

	level := s.roll()
	if level > s.height {
		for i := s.height + 1; i <= level; i++ {
			update[i] = s.head
		}
		s.height = level
	}
	newNode := NewNode(key, value, level)
	for i := 0; i <= level; i++ {
		newNode.next[i] = update[i].next[i]
		update[i].next[i] = newNode
	}
	s.size++
}

// Delete logically deletes key by marking it as a tombstone. The node
// stays in the list — it is only ever removed by a compaction elsewhere
// in the system, never by this call.
func (s *SkipList) Delete(key string) bool {
	current := s.head
	for i := s.height; i >= 0; i-- {
		for current.next[i] != nil && current.next[i].Key < key {
			current = current.next[i]
		}
	}
	target := current.next[0]
	if target != nil && target.Key == key && !target.Deleted {
		target.Deleted = true
		target.Value = nil
		s.size--
		return true
	}
	return false
}

// GetAllSorted returns every node in key order, including tombstones,
// so a Memtable flush can carry deletions into the resulting SSTable.
func (s *SkipList) GetAllSorted() []*Node {
	var result []*Node
	current := s.head.next[0]
	for current != nil {
		result = append(result, current)
		current = current.next[0]
	}
	return result
}
