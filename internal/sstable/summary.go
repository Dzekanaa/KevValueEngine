package sstable

import "encoding/binary"

// summary holds the key range of an SSTable, used to quickly rule out
// a table during a lookup without consulting its Index.
type summary struct {
	MinKey string
	MaxKey string
}

// serialize encodes the summary as a single blob.
func (s summary) serialize() []byte {
	minBytes := []byte(s.MinKey)
	maxBytes := []byte(s.MaxKey)

	buf := make([]byte, 8+len(minBytes)+8+len(maxBytes))
	binary.BigEndian.PutUint64(buf[0:8], uint64(len(minBytes)))
	copy(buf[8:], minBytes)

	tail := 8 + len(minBytes)
	binary.BigEndian.PutUint64(buf[tail:tail+8], uint64(len(maxBytes)))
	copy(buf[tail+8:], maxBytes)

	return buf
}

// deserializeSummary decodes a summary blob.
func deserializeSummary(data []byte) summary {
	minSize := binary.BigEndian.Uint64(data[0:8])
	minKey := string(data[8 : 8+minSize])

	tail := 8 + minSize
	maxSize := binary.BigEndian.Uint64(data[tail : tail+8])
	maxKey := string(data[tail+8 : tail+8+maxSize])

	return summary{MinKey: minKey, MaxKey: maxKey}
}

// inRange reports whether key could belong to a table with this
// summary's key range.
func (s summary) inRange(key string) bool {
	return key >= s.MinKey && key <= s.MaxKey
}
