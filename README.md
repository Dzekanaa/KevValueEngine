
# KevValueEngine

    KevValueEngine/
    |-- cmd/
    |   `-- kvengine/              # main.go — entry point, CLI application
    |-- internal/
    |   |-- wal/                   # Write-Ahead Log — segmented log with CRC integrity checks
    |   |-- memtable/              # Memtable — in-memory hash map structure
    |   |-- sstable/               # SSTable — Data, Filter, Index, Summary
    |   |-- blockmanager/          # Block Manager + Block Cache — only layer that reads/writes files
    |   |-- bloomfilter/           # Bloom Filter implementation
    |   |-- config/                # External configuration loading and parsing
    |   `-- engine/                # Orchestration — combines WAL + Memtable + SSTable for PUT/GET/DELETE
    |-- configs/
    |   `-- config.json            # Configuration file (block, memtable, cache sizes, paths)
    |-- data/                      # Runtime directory for WAL/SSTable files (excluded from git)
    |-- go.mod
    `-- .gitignore

# Go Comment Conventions

- Use `//` for all comments, including doc comments
- Doc comments go directly above the declaration, no blank line in between.
- Doc comments start with the exact name of the identifier they document.
- One doc comment per exported type/function/const/var.
- Package-level doc comment starts with `// Package <name> ...`.
- Inline comments explain *why*, not *what* — skip comments that just restate the code.
- Use `// TODO: ...` for unfinished work.

## Example

```go
// BlockManager reads and writes fixed-size blocks to disk.
// It is the only layer allowed to access files directly.
type BlockManager struct {
	blockSize int
}

// ReadBlock loads the block with the given number from the file.
// Returns an error if the block does not exist or the read fails.
func (bm *BlockManager) ReadBlock(path string, blockNum int) ([]byte, error) {
	// ...
}
```
