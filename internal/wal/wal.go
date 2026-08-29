// Package wal implements a segmented, append-only Write-Ahead Log
// with CRC-verified records, used to recover in-memory state after a crash.
package wal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

// WAL is a segmented, append-only write-ahead log. Records are written
// through a BlockManager, split across fixed-size segment files.
type WAL struct {
	bm                BlockManager
	dir               string
	blockSize         int
	recordsPerSegment int

	currentSegment   int
	recordsInSegment int

	blockNum int
	offset   int
}

// NewWAL creates a WAL that writes segmented files inside dir, using bm
// for all disk access and blockSize-sized blocks with recordsPerSegment
// records per segment.
func NewWAL(dir string, bm BlockManager, blockSize int, recordsPerSegment int) *WAL {
	return &WAL{
		bm:                bm,
		dir:               dir,
		blockSize:         blockSize,
		recordsPerSegment: recordsPerSegment,
		currentSegment:    1,
		recordsInSegment:  0,
		blockNum:          0,
		offset:            0,
	}
}

// Write serializes a record and appends it through the BlockManager,
// rolling over to a new block or segment when the current one is full.
func (w *WAL) Write(key []byte, value []byte, tombstone bool) error {
	record := NewRecord(key, value, tombstone)
	serialized := record.Serialize()

	if len(serialized) > w.blockSize {
		// fragmentation across blocks is not implemented yet
		return errors.New("record too large for a single block")
	}

	if w.recordsInSegment >= w.recordsPerSegment {
		w.currentSegment++
		w.recordsInSegment = 0
		w.blockNum = 0
		w.offset = 0
	}

	path := w.segmentPath(w.currentSegment)

	if w.offset+len(serialized) > w.blockSize {
		w.blockNum++
		w.offset = 0
	}

	currentBlock, err := w.bm.ReadBlock(path, w.blockNum)
	if err != nil {
		// block does not exist yet - start with an empty one
		currentBlock = make([]byte, w.blockSize)
	}

	copy(currentBlock[w.offset:], serialized)

	if err := w.bm.WriteBlock(path, w.blockNum, currentBlock); err != nil {
		return err
	}

	w.offset += len(serialized)
	w.recordsInSegment++

	return nil
}

// Recover reads every record from every segment (oldest first) and
// repositions the WAL's write cursor to resume exactly where the log
// left off, so a subsequent Write does not overwrite existing data.
// It should be called once at startup, before any writes, and is used
// to rebuild the Memtable on startup.
func (w *WAL) Recover() ([]*Record, error) {
	var records []*Record
	segmentNum := 1
	foundAny := false

	for {
		segRecords, lastBlock, lastOffset, recCount, err := w.readSegment(segmentNum)
		if err != nil {
			break
		}
		records = append(records, segRecords...)
		w.currentSegment = segmentNum
		w.blockNum = lastBlock
		w.offset = lastOffset
		w.recordsInSegment = recCount
		foundAny = true
		segmentNum++
	}

	if !foundAny {
		w.currentSegment = 1
		w.blockNum = 0
		w.offset = 0
		w.recordsInSegment = 0
	}

	return records, nil
}

// ReadAll reads every record from every WAL segment, oldest first,
// without changing the write cursor. Use Recover instead if the WAL
// will keep writing afterward.
func (w *WAL) ReadAll() ([]*Record, error) {
	var records []*Record

	segmentNum := 1
	for {
		segRecords, _, _, _, err := w.readSegment(segmentNum)
		if err != nil {
			break
		}
		records = append(records, segRecords...)
		segmentNum++
	}

	return records, nil
}

// readSegment reads every record from a single segment, block by block,
// and reports where writing should resume within it: the last block
// number touched and the offset right after its last real record.
// It returns an error if the segment's first block does not exist.
func (w *WAL) readSegment(segmentNum int) (records []*Record, lastBlock int, lastOffset int, recordCount int, err error) {
	path := w.segmentPath(segmentNum)
	blockNum := 0

	for {
		block, readErr := w.bm.ReadBlock(path, blockNum)
		if readErr != nil {
			if blockNum == 0 {
				return nil, 0, 0, 0, readErr
			}
			break
		}

		offset := 0
		for offset < len(block) {
			remaining := block[offset:]
			if len(remaining) < KEY_START || isAllZero(remaining[:KEY_START]) {
				break
			}

			keySize := binary.BigEndian.Uint64(remaining[KEY_SIZE_START:VALUE_SIZE_START])
			valueSize := binary.BigEndian.Uint64(remaining[VALUE_SIZE_START:KEY_START])
			recordLen := KEY_START + int(keySize) + int(valueSize)

			if offset+recordLen > len(block) {
				break
			}

			record, decErr := Deserialize(remaining[:recordLen])
			if decErr != nil {
				return nil, 0, 0, 0, decErr
			}
			records = append(records, record)
			recordCount++
			offset += recordLen
		}

		lastBlock = blockNum
		lastOffset = offset
		blockNum++
	}

	return records, lastBlock, lastOffset, recordCount, nil
}

// isAllZero reports whether every byte in data is zero, used to detect
// padding written at the end of a block.
func isAllZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// Close is a no-op for now, since the WAL no longer owns a file handle
// directly (all disk access goes through the BlockManager). It exists
// so callers can still call wal.Close() without special-casing it.
func (w *WAL) Close() error {
	return nil
}

// segmentPath returns the file path for the given segment number,
// e.g. wal_0001.log.
func (w *WAL) segmentPath(segmentNum int) string {
	return fmt.Sprintf("%s/wal_%04d.log", w.dir, segmentNum)
}

// DeleteSegment removes a segment file from disk. It refuses to delete
// the currently active segment, since that would discard un-flushed data.
func (w *WAL) DeleteSegment(segmentNum int) error {
	if segmentNum >= w.currentSegment {
		return errors.New("cannot delete the currently active segment")
	}
	return os.Remove(w.segmentPath(segmentNum))
}
