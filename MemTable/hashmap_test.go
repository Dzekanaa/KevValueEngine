package memtable

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPut(t *testing.T) {
	h := NewHashMap()

	entry := Entry{
		Key:   []byte("test"),
		Value: []byte("value"),
	}

	// entry2 := Entry{
	// 	Key:   []byte("test2"),
	// 	Value: []byte("value2"),
	// }

	h.Put(entry)
	fentry, found := h.Get(entry.Key)
	t.Log("Key:\t", string(fentry.Key))
	t.Log("Value:\t", string(fentry.Value))

	t.Logf("state: %+v", h)

	fentry, found = h.Get(entry.Key)
	if found == false {
		t.Fatal("expected to find entry, got nil")
	} else if !bytes.Equal(fentry.Value, entry.Value) {
		t.Fatal("value mismatch")
	} else {
		fmt.Println("Uspeh")
	}

	//fmt.Println()

	// fentry, found = h.Get(entry2.Key)
	// if found == false {
	// 	t.Fatal("expected to find entry2, got nil")
	// } else if !bytes.Equal(fentry.Value, entry2.Value) {
	// 	t.Fatal("value mismatch")
	// } else {
	// 	fmt.Println("Uspeh")
	// }
}
