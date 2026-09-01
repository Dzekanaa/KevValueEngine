package engine

// Get returns the value for key, checking the Memtable first and
// falling back to the SSTables if it's not found there. A key marked
// as deleted (tombstone) in the Memtable is reported as not found,
// without falling through to the SSTables.
func (e *Engine) Get(key string) (value []byte, found bool) {
	if val, found, tombstone := e.memtable.Get(key); found {
		if tombstone {
			return nil, false
		}
		return val, true
	}

	return e.reader.Get(key)
}
