package engine

// Put writes key through the WAL, then applies it to the Memtable,
// flushing first if the Memtable is full.
//
// TODO: implement.
func (e *Engine) Put(key string, value []byte) error {
	return nil
}
