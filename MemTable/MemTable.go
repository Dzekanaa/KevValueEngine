package memtable

const MemtableSize = 10000

type Entry struct {
	Key       []byte
	Value     []byte
	Tombstone bool
}

type MemTable interface {
	Put(entry *Entry)

	Get(key *[]byte)
}
