package memtable

// MockSSTableWriter is a test double for SSTableWriter
type MockSSTableWriter struct {
	WriteCalled  bool
	WriteError   error
	WriteID      string
	WriteEntries []SortedEntry
}

func (m *MockSSTableWriter) Write(entries []SortedEntry) (id string, err error) {
	m.WriteCalled = true
	m.WriteEntries = make([]SortedEntry, len(entries))
	copy(m.WriteEntries, entries)

	if m.WriteError != nil {
		return "", m.WriteError
	}

	return m.WriteID, nil
}
