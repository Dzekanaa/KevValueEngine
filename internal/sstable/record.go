package sstable

import (
	"encoding/binary"
	"errors"
)

// Data record layout on disk:
//
//	+-----------+-----------+-------------+-----+-------+
//	| Tomb (1B) | KeySz(8B) | ValueSz(8B) | Key | Value |
//	+-----------+-----------+-------------+-----+-------+
const (
	tombstoneSize = 1
	keySizeSize   = 8
	valueSizeSize = 8

	tombstoneStart = 0
	keySizeStart   = tombstoneStart + tombstoneSize
	valueSizeStart = keySizeStart + keySizeSize
	dataKeyStart   = valueSizeStart + valueSizeSize
)

// serializeDataRecord encodes a single Data record.
func serializeDataRecord(tombstone bool, key, value []byte) []byte {
	buf := make([]byte, dataKeyStart+len(key)+len(value))

	if tombstone {
		buf[tombstoneStart] = 1
	}
	binary.BigEndian.PutUint64(buf[keySizeStart:valueSizeStart], uint64(len(key)))
	binary.BigEndian.PutUint64(buf[valueSizeStart:dataKeyStart], uint64(len(value)))
	copy(buf[dataKeyStart:], key)
	copy(buf[dataKeyStart+len(key):], value)

	return buf
}

// deserializeDataRecord decodes a Data record starting at the given
// offset within block, returning the tombstone flag and value.
func deserializeDataRecord(block []byte, offset int) (tombstone bool, value []byte, err error) {
	if offset+dataKeyStart > len(block) {
		return false, nil, errors.New("sstable: data record header out of bounds")
	}

	tombstone = block[offset+tombstoneStart] != 0
	keySize := binary.BigEndian.Uint64(block[offset+keySizeStart : offset+valueSizeStart])
	valueSize := binary.BigEndian.Uint64(block[offset+valueSizeStart : offset+dataKeyStart])

	valueStart := offset + dataKeyStart + int(keySize)
	valueEnd := valueStart + int(valueSize)
	if valueEnd > len(block) {
		return false, nil, errors.New("sstable: data record value out of bounds")
	}

	value = make([]byte, valueSize)
	copy(value, block[valueStart:valueEnd])

	return tombstone, value, nil
}
