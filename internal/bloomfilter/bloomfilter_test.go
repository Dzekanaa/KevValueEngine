package bloomfilter

import (
	"testing"
)

func TestBloomFilter(t *testing.T) {
	n := 100
	p := 0.01

	m := CalculateM(n, p)
	k := CalculateK(n, m)

	hashFns := CreateHashFunctions(uint32(k))
	bf := NewBloomFilter(m, k, hashFns)

	data := []byte("test-kljuc")
	hashes := GetHashes(data, hashFns)
	bf.Add(hashes)

	if !bf.Contains(hashes) {
		t.Error("expected filter to contain the added key")
	}

	data2 := []byte("nepostojeci-kljuc")
	hashes2 := GetHashes(data2, hashFns)
	if bf.Contains(hashes2) {
		t.Log("false positive on absent key (expected occasionally)")
	}
}

func TestSerializeDeserialize(t *testing.T) {
	n := 100
	p := 0.01
	m := CalculateM(n, p)
	k := CalculateK(n, m)
	hashFns := CreateHashFunctions(uint32(k))
	bf := NewBloomFilter(m, k, hashFns)

	data := []byte("test-kljuc")
	hashes := GetHashes(data, hashFns)
	bf.Add(hashes)

	serialized := bf.Serialize()
	bf2, err := Deserialize(serialized)
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	if bf2.M != bf.M {
		t.Errorf("M mismatch: expected %d, got %d", bf.M, bf2.M)
	}
	if bf2.K != bf.K {
		t.Errorf("K mismatch: expected %d, got %d", bf.K, bf2.K)
	}
	if len(bf2.Bitset) != len(bf.Bitset) {
		t.Fatalf("bitset length mismatch: expected %d, got %d", len(bf.Bitset), len(bf2.Bitset))
	}
	if len(bf2.HashFns) != len(bf.HashFns) {
		t.Fatalf("hash function count mismatch: expected %d, got %d", len(bf.HashFns), len(bf2.HashFns))
	}

	if !bf2.Contains(hashes) {
		t.Error("deserialized filter does not recognize a previously added key")
	}

	data2 := []byte("nepostojeci-kljuc-xyz")
	hashes2 := GetHashes(data2, hashFns)
	if bf.Contains(hashes2) != bf2.Contains(hashes2) {
		t.Error("original and deserialized filter disagree on an absent key")
	}
}

func TestMerge(t *testing.T) {
	n := 100
	p := 0.01
	m := CalculateM(n, p)
	k := CalculateK(n, m)
	hashFns := CreateHashFunctions(uint32(k))

	bf1 := NewBloomFilter(m, k, hashFns)
	bf2 := NewBloomFilter(m, k, hashFns)

	data1 := []byte("kljuc-1")
	data2 := []byte("kljuc-2")

	bf1.Add(GetHashes(data1, hashFns))
	bf2.Add(GetHashes(data2, hashFns))

	if err := bf1.Merge(bf2); err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	if !bf1.Contains(GetHashes(data1, hashFns)) {
		t.Error("merged filter should still contain data1")
	}
	if !bf1.Contains(GetHashes(data2, hashFns)) {
		t.Error("merged filter should now contain data2 from bf2")
	}
}

func TestMergeDifferentParams(t *testing.T) {
	bf1 := NewBloomFilter(100, 3, CreateHashFunctions(3))
	bf2 := NewBloomFilter(200, 5, CreateHashFunctions(5))

	if err := bf1.Merge(bf2); err == nil {
		t.Error("expected error when merging filters with different parameters")
	}
}
