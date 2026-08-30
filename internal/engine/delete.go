package engine

// Delete marks key as deleted, through the WAL and then the Memtable,
// flushing first if the Memtable is full.
//
// TODO: implement.
func (e *Engine) Delete(key string) error {
	return nil
}
