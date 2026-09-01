package sstable

import "github.com/Dzekanaa/KevValueEngine/internal/bloomfilter"

// bloomFalsePositiveRate is the target false-positive rate used when
// sizing a new Bloom filter for an SSTable.
const bloomFalsePositiveRate = 0.01

// newBloomFilter builds a BloomFilter sized for len(keys) entries at
// the target false-positive rate, and adds every key to it.
func newBloomFilter(keys []string) *bloomfilter.BloomFilter {
	m := bloomfilter.CalculateM(len(keys), bloomFalsePositiveRate)
	k := bloomfilter.CalculateK(len(keys), m)
	hashFns := bloomfilter.CreateHashFunctions(uint32(k))

	bf := bloomfilter.NewBloomFilter(m, k, hashFns)
	for _, key := range keys {
		bf.Add(bloomfilter.GetHashes([]byte(key), bf.HashFns))
	}

	return bf
}

// bloomMightContain reports whether key might be present in bf. A true
// result may be a false positive; a false result is always accurate.
// It must use bf.HashFns (the seeds the filter was actually built or
// loaded with), never a freshly generated set — otherwise Contains
// would check the wrong bits entirely.
func bloomMightContain(bf *bloomfilter.BloomFilter, key string) bool {
	return bf.Contains(bloomfilter.GetHashes([]byte(key), bf.HashFns))
}
