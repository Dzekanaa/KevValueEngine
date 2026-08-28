package merkle

import "testing"

func TestNewBuildsTreeWithoutError(t *testing.T) {
	blocks := [][]byte{[]byte("A"), []byte("B"), []byte("C"), []byte("D"), []byte("E")}

	mr, err := New(blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mr.String() == "" {
		t.Fatal("expected a non-empty root hash")
	}
}

func TestSerializeDeserializeRoundTrip(t *testing.T) {
	blocks := [][]byte{[]byte("A"), []byte("B"), []byte("C"), []byte("D"), []byte("E")}

	original, err := New(blocks)
	if err != nil {
		t.Fatalf("unexpected error building tree: %v", err)
	}

	data := original.Serialize()

	restored, err := Deserialize(data)
	if err != nil {
		t.Fatalf("unexpected error deserializing: %v", err)
	}

	if original.String() != restored.String() {
		t.Fatalf("root hash mismatch: original=%s, restored=%s", original.String(), restored.String())
	}
}

func TestSerializeDeserializePreservesLeafInfo(t *testing.T) {
	blocks := [][]byte{[]byte("A"), []byte("B"), []byte("C")}

	original, err := New(blocks)
	if err != nil {
		t.Fatalf("unexpected error building tree: %v", err)
	}

	restored, err := Deserialize(original.Serialize())
	if err != nil {
		t.Fatalf("unexpected error deserializing: %v", err)
	}

	if !restored.root.left.left.leaf {
		t.Fatal("expected a leaf at root.left.left")
	}
	if restored.root.left.left.index != 0 {
		t.Fatalf("expected index 0, got %d", restored.root.left.left.index)
	}
}

func TestCompareDetectsChangedBlock(t *testing.T) {
	oldBlocks := [][]byte{[]byte("A"), []byte("B"), []byte("C"), []byte("D")}
	newBlocks := [][]byte{[]byte("A"), []byte("CHANGED"), []byte("C"), []byte("D")}

	oldTree, err := New(oldBlocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	newTree, err := New(newBlocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	diffs := Compare(oldTree, newTree)
	if len(diffs) != 1 || diffs[0] != 1 {
		t.Fatalf("expected exactly one changed block at index 1, got: %v", diffs)
	}
}

func TestCompareNoChangesMeansEmpty(t *testing.T) {
	blocks := [][]byte{[]byte("A"), []byte("B"), []byte("C")}

	t1, err := New(blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t2, err := New(blocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	diffs := Compare(t1, t2)
	if len(diffs) != 0 {
		t.Fatalf("expected no differences, got: %v", diffs)
	}
}
