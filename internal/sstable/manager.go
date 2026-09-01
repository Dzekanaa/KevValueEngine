// Package sstable implements the on-disk SSTable structure: Data,
// Filter, Index, and Summary files, plus lookups across all existing
// SSTables (newest first).
package sstable

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/Dzekanaa/KevValueEngine/internal/bloomfilter"
	"github.com/Dzekanaa/KevValueEngine/internal/memtable"
)

// Manager creates SSTables from flushed Memtables and serves lookups
// across all SSTables created so far. It implements both
// engine.SSTableWriter and engine.SSTableReader.
type Manager struct {
	mu        sync.RWMutex
	bm        BlockManager
	dataDir   string
	blockSize int
	nextID    int
	tables    []string // creation order, oldest first
}

// NewManager creates a Manager that persists SSTables into dataDir,
// using bm for all disk access with the given block size.
func NewManager(bm BlockManager, dataDir string, blockSize int) *Manager {
	return &Manager{
		bm:        bm,
		dataDir:   dataDir,
		blockSize: blockSize,
		nextID:    1,
	}
}

// Write persists entries (already sorted by key) as a new SSTable:
// Data, Filter, Index, and Summary, in that order.
func (m *Manager) Write(entries []memtable.SortedEntry) (string, error) {
	if len(entries) == 0 {
		return "", fmt.Errorf("sstable: cannot write an empty entry set")
	}

	m.mu.Lock()
	id := fmt.Sprintf("sstable_%04d", m.nextID)
	m.nextID++
	m.mu.Unlock()

	dataPath := m.path(id, "data")
	filterPath := m.path(id, "filter")
	indexPath := m.path(id, "index")
	summaryPath := m.path(id, "summary")

	index, err := writeData(m.bm, dataPath, m.blockSize, entries)
	if err != nil {
		return "", fmt.Errorf("sstable %s: write data: %w", id, err)
	}

	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.Key
	}
	bf := newBloomFilter(keys)
	if err := writeBlob(m.bm, filterPath, m.blockSize, bf.Serialize()); err != nil {
		return "", fmt.Errorf("sstable %s: write filter: %w", id, err)
	}
	if err := writeIndex(m.bm, indexPath, m.blockSize, index); err != nil {
		return "", fmt.Errorf("sstable %s: write index: %w", id, err)
	}

	sum := summary{MinKey: entries[0].Key, MaxKey: entries[len(entries)-1].Key}
	if err := writeBlob(m.bm, summaryPath, m.blockSize, sum.serialize()); err != nil {
		return "", fmt.Errorf("sstable %s: write summary: %w", id, err)
	}

	m.mu.Lock()
	m.tables = append(m.tables, id)
	m.mu.Unlock()

	return id, nil
}

// Get searches all known SSTables for key, newest first, using each
// table's Filter to skip tables that can't contain it, then its
// Summary and Index to locate the value in Data. found is false if
// the key isn't in any SSTable, or is present only as a tombstone.
func (m *Manager) Get(key string) ([]byte, bool) {
	m.mu.RLock()
	tables := make([]string, len(m.tables))
	copy(tables, m.tables)
	m.mu.RUnlock()

	for i := len(tables) - 1; i >= 0; i-- {
		id := tables[i]

		value, found, isTombstone, err := m.getFromTable(id, key)
		if err != nil {
			continue // TODO: surface/log this instead of silently skipping
		}
		if !found {
			continue
		}
		if isTombstone {
			return nil, false // deleted — don't look at older tables
		}
		return value, true
	}

	return nil, false
}

// getFromTable looks up key within a single SSTable.
func (m *Manager) getFromTable(id, key string) (value []byte, found bool, tombstone bool, err error) {
	filterData, err := readBlob(m.bm, m.path(id, "filter"), m.blockSize)
	if err != nil {
		return nil, false, false, err
	}
	bf, err := bloomfilter.Deserialize(filterData)
	if err != nil {
		return nil, false, false, err
	}
	if !bloomMightContain(bf, key) {
		return nil, false, false, nil
	}

	summaryData, err := readBlob(m.bm, m.path(id, "summary"), m.blockSize)
	if err != nil {
		return nil, false, false, err
	}
	sum := deserializeSummary(summaryData)
	if !sum.inRange(key) {
		return nil, false, false, nil
	}

	blockNum, offset, found, err := readIndex(m.bm, m.path(id, "index"), key)
	if err != nil {
		return nil, false, false, err
	}
	if !found {
		return nil, false, false, nil
	}

	tombstone, value, err = readDataRecord(m.bm, m.path(id, "data"), blockNum, offset)
	if err != nil {
		return nil, false, false, err
	}

	return value, true, tombstone, nil
}

// path builds the file path for a given SSTable component.
func (m *Manager) path(id, component string) string {
	return filepath.Join(m.dataDir, fmt.Sprintf("%s_%s.bin", id, component))
}
