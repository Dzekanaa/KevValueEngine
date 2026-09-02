package sstable

// BlockManager is the interface the sstable package uses for all disk
// access, matching the same contract used by wal.
type BlockManager interface {
	ReadBlock(path string, blockNum int) ([]byte, error)
	WriteBlock(path string, blockNum int, data []byte) error
}
