
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

