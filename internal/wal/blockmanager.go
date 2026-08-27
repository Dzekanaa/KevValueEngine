package wal

// BlockManager is the interface the WAL uses for all disk access.
// It is implemented by the real block manager package; the WAL only
// depends on this contract, never on that package directly.
type BlockManager interface {
	ReadBlock(path string, blockNum int) ([]byte, error)
	WriteBlock(path string, blockNum int, data []byte) error
}
