package engine

// Recover rebuilds the Memtable from the WAL on startup, replaying
// every record in order (Put for normal records, Delete for tombstones).
//
// Marija i Niksa
// TODO: implement this once Flush is working
// end to end.
func (e *Engine) Recover() error {
	return nil
}
