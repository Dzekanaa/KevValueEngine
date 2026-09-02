package sstable

import (
	"fmt"

	"github.com/Dzekanaa/KevValueEngine/internal/memtable"
)

// indexEntry records where a single key's Data record was written,
// used to build the Index file.
type indexEntry struct {
	Key      string
	BlockNum int
	Offset   int
}

// writeData serializes entries (already sorted by key) into the Data
// file at path, packing records into blockSize-sized blocks, and
// returns the position of each key for building the Index.
func writeData(bm BlockManager, path string, blockSize int, entries []memtable.SortedEntry) ([]indexEntry, error) {
	var index []indexEntry

	blockNum := 0
	offset := 0
	block := make([]byte, blockSize)

	flush := func() error {
		return bm.WriteBlock(path, blockNum, block)
	}

	for _, e := range entries {
		record := serializeDataRecord(e.Tombstone, []byte(e.Key), e.Value)
		if len(record) > blockSize {
			return nil, fmt.Errorf("sstable: data record for key %q exceeds block size", e.Key)
		}

		if offset+len(record) > blockSize {
			if err := flush(); err != nil {
				return nil, err
			}
			blockNum++
			offset = 0
			block = make([]byte, blockSize)
		}

		index = append(index, indexEntry{Key: e.Key, BlockNum: blockNum, Offset: offset})
		copy(block[offset:], record)
		offset += len(record)
	}

	if offset > 0 {
		if err := flush(); err != nil {
			return nil, err
		}
	}

	return index, nil
}

// readDataRecord reads a single Data record at the given block and
// offset, as located by an Index lookup.
func readDataRecord(bm BlockManager, path string, blockNum, offset int) (tombstone bool, value []byte, err error) {
	block, err := bm.ReadBlock(path, blockNum)
	if err != nil {
		return false, nil, fmt.Errorf("sstable: read data block %d: %w", blockNum, err)
	}
	return deserializeDataRecord(block, offset)
}
