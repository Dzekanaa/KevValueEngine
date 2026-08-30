package engine

// Engine ties together the WAL, Memtable, and SSTable reader/writer to
// implement the system's PUT/GET/DELETE operations, flush, and
// startup recovery.
type Engine struct {
	wal      WAL
	memtable Memtable
	writer   SSTableWriter
	reader   SSTableReader
}

// New creates an Engine wired to the given WAL, Memtable, SSTable
// writer, and SSTable reader implementations.
func New(wal WAL, memtable Memtable, writer SSTableWriter, reader SSTableReader) *Engine {
	return &Engine{
		wal:      wal,
		memtable: memtable,
		writer:   writer,
		reader:   reader,
	}
}
