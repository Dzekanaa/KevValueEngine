package sstable

import (
	"encoding/binary"
	"sort"
)

// Index record layout on disk:
//
//	+-----------+-----+---------------+-------------+
//	| KeySz(8B) | Key | BlockNum (8B) | Offset (8B) |
//	+-----------+-----+---------------+-------------+
//
// KeySize == 0 marks the start of zero-padding at the end of a block.
const (
	idxKeySizeSize = 8
	idxBlockSize   = 8
	idxOffsetSize  = 8
)

// serializeIndexRecord encodes a single Index record.
func serializeIndexRecord(key string, blockNum, offset int) []byte {
	keyBytes := []byte(key)
	buf := make([]byte, idxKeySizeSize+len(keyBytes)+idxBlockSize+idxOffsetSize)

	binary.BigEndian.PutUint64(buf[0:idxKeySizeSize], uint64(len(keyBytes)))
	copy(buf[idxKeySizeSize:], keyBytes)

	tail := idxKeySizeSize + len(keyBytes)
	binary.BigEndian.PutUint64(buf[tail:tail+idxBlockSize], uint64(blockNum))
	binary.BigEndian.PutUint64(buf[tail+idxBlockSize:tail+idxBlockSize+idxOffsetSize], uint64(offset))

	return buf
}

// writeIndex serializes entries into the Index file at path, packing
// records into blockSize-sized blocks.
func writeIndex(bm BlockManager, path string, blockSize int, entries []indexEntry) error {
	blockNum := 0
	offset := 0
	block := make([]byte, blockSize)

	flush := func() error {
		return bm.WriteBlock(path, blockNum, block)
	}

	for _, e := range entries {
		record := serializeIndexRecord(e.Key, e.BlockNum, e.Offset)

		if offset+len(record) > blockSize {
			if err := flush(); err != nil {
				return err
			}
			blockNum++
			offset = 0
			block = make([]byte, blockSize)
		}

		copy(block[offset:], record)
		offset += len(record)
	}

	if offset > 0 {
		if err := flush(); err != nil {
			return err
		}
	}

	return nil
}

// readIndex reads every Index record from path, block by block, and
// returns the position of key within it, if present. Entries are
// assumed sorted by key (true for tables built from writeData), so
// this performs a binary search rather than a linear scan.
func readIndex(bm BlockManager, path string, key string) (blockNum, offset int, found bool, err error) {
	entries, err := readAllIndexEntries(bm, path)
	if err != nil {
		return 0, 0, false, err
	}

	i := sort.Search(len(entries), func(i int) bool {
		return entries[i].Key >= key
	})

	if i < len(entries) && entries[i].Key == key {
		return entries[i].BlockNum, entries[i].Offset, true, nil
	}

	return 0, 0, false, nil
}

// readAllIndexEntries reads and parses every Index record from path.
func readAllIndexEntries(bm BlockManager, path string) ([]indexEntry, error) {
	var entries []indexEntry
	var pending []byte

	blockNum := 0
	for {
		block, err := bm.ReadBlock(path, blockNum)
		if err != nil {
			if blockNum == 0 {
				return nil, err
			}
			break
		}
		pending = append(pending, block...)

		for len(pending) >= idxKeySizeSize {
			keySize := binary.BigEndian.Uint64(pending[:idxKeySizeSize])
			if keySize == 0 {
				pending = nil // rest of block is padding
				break
			}

			recordLen := idxKeySizeSize + int(keySize) + idxBlockSize + idxOffsetSize
			if len(pending) < recordLen {
				break
			}

			key := string(pending[idxKeySizeSize : idxKeySizeSize+int(keySize)])
			tail := idxKeySizeSize + int(keySize)
			blockNumEntry := binary.BigEndian.Uint64(pending[tail : tail+idxBlockSize])
			offsetEntry := binary.BigEndian.Uint64(pending[tail+idxBlockSize : tail+idxBlockSize+idxOffsetSize])

			entries = append(entries, indexEntry{
				Key:      key,
				BlockNum: int(blockNumEntry),
				Offset:   int(offsetEntry),
			})

			pending = pending[recordLen:]
		}

		blockNum++
	}

	return entries, nil
}
