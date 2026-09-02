package skiplist

import (
	"bytes"
	"testing"
)

func TestSkipListBasicOperations(t *testing.T) {
	s := NewSkipList(4)

	key := "test_key"
	val := []byte("test_value")

	s.Insert(key, val)
	if s.size != 1 {
		t.Fatalf("expected size 1, got %d", s.size)
	}

	fval, found := s.Get(key)
	if !found {
		t.Fatal("expected to find entry")
	}
	if !bytes.Equal(fval, val) {
		t.Fatal("value mismatch")
	}

	if !s.Delete(key) {
		t.Fatal("expected successful delete")
	}
	if _, found := s.Get(key); found {
		t.Fatal("entry should be logically deleted")
	}
}

func TestSkipListReinsertAfterDelete(t *testing.T) {
	s := NewSkipList(4)
	key := "key"

	s.Insert(key, []byte("v1"))
	s.Delete(key)
	if s.size != 0 {
		t.Fatalf("expected size 0 after delete, got %d", s.size)
	}

	s.Insert(key, []byte("v2"))
	if s.size != 1 {
		t.Fatalf("expected size 1 after reinsert, got %d", s.size)
	}

	val, found := s.Get(key)
	if !found || !bytes.Equal(val, []byte("v2")) {
		t.Fatal("expected reinserted value to be found")
	}
}

func TestSkipListGetAllSortedIncludesTombstones(t *testing.T) {
	s := NewSkipList(4)
	s.Insert("a", []byte("1"))
	s.Insert("b", []byte("2"))
	s.Insert("c", []byte("3"))
	s.Delete("b")

	nodes := s.GetAllSorted()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes including tombstone, got %d", len(nodes))
	}

	found := false
	for _, n := range nodes {
		if n.Key == "b" {
			found = true
			if !n.Deleted {
				t.Fatal("expected node b to be marked deleted")
			}
		}
	}
	if !found {
		t.Fatal("expected tombstone for key b to be present in GetAllSorted")
	}
}
