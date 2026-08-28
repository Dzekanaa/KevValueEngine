package wal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"time"
)

// Record layout on disk:
//
//	+---------------+-----------------+---------------+---------------+-----------------+-...-+--...--+
//	|    CRC (4B)   | Timestamp (8B) | Tombstone(1B) | Key Size (8B) | Value Size (8B) | Key | Value |
//	+---------------+-----------------+---------------+---------------+-----------------+-...-+--...--+
//
// CRC is a 32-bit checksum computed over every field except itself.
// Timestamp is the operation time in seconds.
// Tombstone marks the record as a logical delete.
const (
	CRC_SIZE        = 4
	TIMESTAMP_SIZE  = 8
	TOMBSTONE_SIZE  = 1
	KEY_SIZE_SIZE   = 8
	VALUE_SIZE_SIZE = 8

	CRC_START        = 0
	TIMESTAMP_START  = CRC_START + CRC_SIZE
	TOMBSTONE_START  = TIMESTAMP_START + TIMESTAMP_SIZE
	KEY_SIZE_START   = TOMBSTONE_START + TOMBSTONE_SIZE
	VALUE_SIZE_START = KEY_SIZE_START + KEY_SIZE_SIZE
	KEY_START        = VALUE_SIZE_START + VALUE_SIZE_SIZE
)

// CRC32 computes the checksum used to detect corrupted records.
func CRC32(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// Record is a single WAL entry: a key-value pair plus the metadata
// needed to verify and replay it during recovery.
type Record struct {
	CRC       uint32
	Timestamp int64
	Tombstone bool
	Type      byte // FULL, FIRST, MIDDLE, LAST - reserved for future fragmentation support
	KeySize   uint64
	ValueSize uint64
	Key       []byte
	Value     []byte
}

// NewRecord creates a Record from a key, value and tombstone flag,
// stamping it with the current time. The CRC is left unset until Serialize
// is called, since it depends on the record's serialized form.
func NewRecord(key []byte, value []byte, tombstone bool) *Record {
	return &Record{
		Timestamp: time.Now().Unix(),
		Tombstone: tombstone,
		KeySize:   uint64(len(key)),
		ValueSize: uint64(len(value)),
		Key:       key,
		Value:     value,
	}
}

// Serialize encodes the record into its on-disk byte representation,
// computing and prepending the CRC over the remaining fields.
func (r *Record) Serialize() []byte {
	payload := bytes.Buffer{}
	binary.Write(&payload, binary.BigEndian, r.Timestamp)
	binary.Write(&payload, binary.BigEndian, r.Tombstone)
	binary.Write(&payload, binary.BigEndian, r.KeySize)
	binary.Write(&payload, binary.BigEndian, r.ValueSize)
	payload.Write(r.Key)
	payload.Write(r.Value)

	r.CRC = CRC32(payload.Bytes())

	final := bytes.Buffer{}
	binary.Write(&final, binary.BigEndian, r.CRC)
	final.Write(payload.Bytes())

	return final.Bytes()
}

// Deserialize decodes a Record from its on-disk byte representation.
// It returns an error if the stored CRC does not match the computed one,
// indicating the record is corrupted.
func Deserialize(data []byte) (*Record, error) {
	crc := binary.BigEndian.Uint32(data[CRC_START:TIMESTAMP_START])

	computedCRC := CRC32(data[TIMESTAMP_START:])
	if crc != computedCRC {
		return nil, errors.New("checksum mismatch: record is corrupted")
	}

	timestamp := int64(binary.BigEndian.Uint64(data[TIMESTAMP_START:TOMBSTONE_START]))
	tombstone := data[TOMBSTONE_START] != 0
	keySize := binary.BigEndian.Uint64(data[KEY_SIZE_START:VALUE_SIZE_START])
	valueSize := binary.BigEndian.Uint64(data[VALUE_SIZE_START:KEY_START])

	keyEnd := KEY_START + int(keySize)
	valueEnd := keyEnd + int(valueSize)

	return &Record{
		CRC:       crc,
		Timestamp: timestamp,
		Tombstone: tombstone,
		KeySize:   keySize,
		ValueSize: valueSize,
		Key:       data[KEY_START:keyEnd],
		Value:     data[keyEnd:valueEnd],
	}, nil
}
