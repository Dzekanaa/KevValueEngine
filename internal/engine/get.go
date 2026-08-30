package engine

// Get returns the value for key, checking the Memtable first and
// falling back to the SSTables if it's not found there.
//
// TODO: implement.
func (e *Engine) Get(key string) (value []byte, found bool) {
	return nil, false
}
