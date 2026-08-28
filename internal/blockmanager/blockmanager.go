// Package blockmanager implements the Block Manager with fixed-size block access to disk via mmap,
// always backed by an LRU block cache.
package blockmanager

import (
	"errors"
	"fmt"
	"os"

	"github.com/edsrzf/mmap-go"

	"github.com/Dzekanaa/KevValueEngine/internal/cache"
)

// ErrBlockNotFound is returned when the requested block does not exist -
// either the file doesn't exist yet, or the block lies at or beyond the
// current end of the file.
var ErrBlockNotFound = errors.New("block not found")

// BlockManager reads and writes fixed-size blocks to disk via mmap,
// checking an LRU cache before every read and refreshing it after
// every write. It is the only component allowed to access files
// directly - everything else goes through it.
type BlockManager struct {
	blockSize int
	cache     *cache.LRUCache
}

// New creates the Block Manager: blockSize-sized blocks, backed by an
// LRU cache holding up to cacheCapacity blocks.
func New(blockSize int, cacheCapacity int) *BlockManager {
	return &BlockManager{
		blockSize: blockSize,
		cache:     cache.NewLRUCache(cacheCapacity),
	}
}

// BlockSize returns the configured block size in bytes.
func (bm *BlockManager) BlockSize() int {
	return bm.blockSize
}

// cacheKey builds the cache key identifying a block within a file.
func cacheKey(path string, blockNum int) string {
	return fmt.Sprintf("%s:%d", path, blockNum)
}

// ReadBlock returns the block with the given number from the file at
// path. It checks the cache first; on a miss it reads via mmap from
// disk and populates the cache before returning. It returns
// ErrBlockNotFound if the file doesn't exist yet or the requested
// block lies at or beyond the current end of the file.
func (bm *BlockManager) ReadBlock(path string, blockNum int) ([]byte, error) {
	key := cacheKey(path, blockNum)

	if data, found := bm.cache.Get(key); found {
		return data, nil
	}

	data, err := bm.readFromDisk(path, blockNum)
	if err != nil {
		return nil, err
	}

	bm.cache.Put(key, data)
	return data, nil
}

// readFromDisk reads a block directly from disk via mmap, bypassing
// the cache. Used internally by ReadBlock on a cache miss.
func (bm *BlockManager) readFromDisk(path string, blockNum int) ([]byte, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0o644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBlockNotFound
		}
		return nil, fmt.Errorf("blockmanager: open %s: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("blockmanager: stat %s: %w", path, err)
	}

	offset := int64(blockNum) * int64(bm.blockSize)
	if offset >= info.Size() {
		return nil, ErrBlockNotFound
	}

	mmapFile, err := mmap.Map(file, mmap.RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("blockmanager: mmap %s: %w", path, err)
	}
	defer mmapFile.Unmap()

	block := make([]byte, bm.blockSize)
	end := offset + int64(bm.blockSize)
	if end > int64(len(mmapFile)) {
		end = int64(len(mmapFile))
	}
	copy(block, mmapFile[offset:end])

	return block, nil
}
