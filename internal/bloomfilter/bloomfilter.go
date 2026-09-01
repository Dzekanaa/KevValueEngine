// Package bloomfilter implements a probabilistic set membership structure.
// A BloomFilter can report false positives ("maybe present") but never
// false negatives ("definitely absent"), making it useful as a cheap
// pre-check before an expensive lookup (e.g. scanning an SSTable).
package bloomfilter

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"math"
	"time"
)

// HashWithSeed wraps a seed value used to derive one of the k independent
// hash functions a BloomFilter needs.
type HashWithSeed struct {
	Seed []byte
}

// Hash computes an MD5-based hash of data combined with the seed.
func (h HashWithSeed) Hash(data []byte) uint64 {
	fn := md5.New()
	fn.Write(append(data, h.Seed...))
	return binary.BigEndian.Uint64(fn.Sum(nil))
}

// CreateHashFunctions builds k independent hash functions, each seeded
// with a distinct value derived from the current time.
func CreateHashFunctions(k uint32) []HashWithSeed {
	h := make([]HashWithSeed, k)
	ts := uint32(time.Now().Unix())
	for i := uint32(0); i < k; i++ {
		seed := make([]byte, 4)
		binary.BigEndian.PutUint32(seed, ts+i)
		hfn := HashWithSeed{Seed: seed}
		h[i] = hfn
	}
	return h
}

// CalculateM returns the optimal bitset size for expectedElements items
// at the given falsePositiveRate.
func CalculateM(expectedElements int, falsePositiveRate float64) uint64 {
	return uint64(math.Ceil(float64(expectedElements) * math.Abs(math.Log(falsePositiveRate)) / math.Pow(math.Log(2), float64(2))))
}

// CalculateK returns the optimal number of hash functions for a bitset
// of size m holding expectedElements items.
func CalculateK(expectedElements int, m uint64) uint64 {
	return uint64(math.Ceil((float64(m) / float64(expectedElements)) * math.Log(2)))
}

// BloomFilter is a probabilistic set that supports Add and Contains,
// backed by a bitset and a fixed set of hash functions.
type BloomFilter struct {
	Bitset  []uint8
	M       uint64
	K       uint64
	HashFns []HashWithSeed
}

// NewBloomFilter creates an empty BloomFilter with the given bitset size,
// hash function count, and hash functions.
func NewBloomFilter(m uint64, k uint64, hashFns []HashWithSeed) *BloomFilter {
	return &BloomFilter{
		Bitset:  make([]uint8, m),
		M:       m,
		K:       k,
		HashFns: hashFns,
	}
}

// Add marks the given hashes as present in the filter.
func (bf *BloomFilter) Add(hashes []uint64) {
	for _, hash := range hashes {
		bf.Bitset[hash%bf.M] = 1
	}
}

// Contains reports whether all the given hashes are marked as present.
// A true result may be a false positive; a false result is always accurate.
func (bf *BloomFilter) Contains(hashes []uint64) bool {
	for _, hash := range hashes {
		if bf.Bitset[hash%bf.M] == 0 {
			return false
		}
	}
	return true
}

// GetHashes computes one hash per hash function for the given data.
func GetHashes(data []byte, hashFns []HashWithSeed) []uint64 {
	hashes := make([]uint64, len(hashFns))
	for i, h := range hashFns {
		hashes[i] = h.Hash(data)
	}
	return hashes
}

// Serialize encodes the filter's parameters, hash function seeds, and
// bitset into its on-disk byte representation.
func (bf *BloomFilter) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, bf.M)
	binary.Write(buf, binary.BigEndian, bf.K)

	binary.Write(buf, binary.BigEndian, uint32(len(bf.HashFns)))

	for _, h := range bf.HashFns {
		binary.Write(buf, binary.BigEndian, uint32(len(h.Seed)))
		buf.Write(h.Seed)
	}

	binary.Write(buf, binary.BigEndian, uint64(len(bf.Bitset)))
	buf.Write(bf.Bitset)

	return buf.Bytes()
}

// Deserialize decodes a BloomFilter from its on-disk byte representation.
func Deserialize(data []byte) (*BloomFilter, error) {
	buf := bytes.NewReader(data)
	bf := &BloomFilter{}

	if err := binary.Read(buf, binary.BigEndian, &bf.M); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &bf.K); err != nil {
		return nil, err
	}

	var numHashFns uint32
	if err := binary.Read(buf, binary.BigEndian, &numHashFns); err != nil {
		return nil, err
	}

	bf.HashFns = make([]HashWithSeed, numHashFns)
	for i := uint32(0); i < numHashFns; i++ {
		var seedLen uint32
		if err := binary.Read(buf, binary.BigEndian, &seedLen); err != nil {
			return nil, err
		}
		seed := make([]byte, seedLen)
		if _, err := buf.Read(seed); err != nil {
			return nil, err
		}
		bf.HashFns[i] = HashWithSeed{Seed: seed}
	}

	var bitsetLen uint64
	if err := binary.Read(buf, binary.BigEndian, &bitsetLen); err != nil {
		return nil, err
	}
	bf.Bitset = make([]uint8, bitsetLen)
	if _, err := buf.Read(bf.Bitset); err != nil {
		return nil, err
	}
	return bf, nil
}

// Merge combines two BloomFilters built with identical parameters (M, K)
// into one, via bitwise OR — used when compacting SSTables whose filters
// must be combined without rebuilding from scratch.
func (bf *BloomFilter) Merge(other *BloomFilter) error {
	if bf.M != other.M || bf.K != other.K {
		return errors.New("cannot merge bloom filters with different parameters")
	}
	for i := range bf.Bitset {
		bf.Bitset[i] |= other.Bitset[i]
	}
	return nil
}
