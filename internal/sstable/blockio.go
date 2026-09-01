package sstable

import (
	"encoding/binary"
	"fmt"
)

// writeBlob writes data as a single length-prefixed blob, packed into
// blockSize-sized blocks. Used for Filter and Summary, which are
// single binary blobs rather than sequences of records.
func writeBlob(bm BlockManager, path string, blockSize int, data []byte) error {
	prefixed := make([]byte, 8+len(data))
	binary.BigEndian.PutUint64(prefixed[:8], uint64(len(data)))
	copy(prefixed[8:], data)

	blockNum := 0
	for written := 0; written < len(prefixed); {
		block := make([]byte, blockSize)
		n := copy(block, prefixed[written:])
		if err := bm.WriteBlock(path, blockNum, block); err != nil {
			return fmt.Errorf("sstable: write blob block %d: %w", blockNum, err)
		}
		written += n
		blockNum++
	}

	return nil
}

// readBlob reads back a blob written by writeBlob.
func readBlob(bm BlockManager, path string, blockSize int) ([]byte, error) {
	first, err := bm.ReadBlock(path, 0)
	if err != nil {
		return nil, err
	}
	if len(first) < 8 {
		return nil, fmt.Errorf("sstable: blob at %s is too short to contain a length prefix", path)
	}

	length := binary.BigEndian.Uint64(first[:8])
	total := 8 + int(length)

	data := make([]byte, 0, total)
	data = append(data, first...)

	blockNum := 1
	for len(data) < total {
		block, err := bm.ReadBlock(path, blockNum)
		if err != nil {
			return nil, fmt.Errorf("sstable: read blob block %d: %w", blockNum, err)
		}
		data = append(data, block...)
		blockNum++
	}

	return data[8 : 8+length], nil
}
