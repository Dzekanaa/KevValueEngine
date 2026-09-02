package main

import (
	"fmt"
	"os"

	"github.com/Dzekanaa/KevValueEngine/internal/blockmanager"
	"github.com/Dzekanaa/KevValueEngine/internal/config"
	"github.com/Dzekanaa/KevValueEngine/internal/engine"
	"github.com/Dzekanaa/KevValueEngine/internal/memtable"
	"github.com/Dzekanaa/KevValueEngine/internal/sstable"
	"github.com/Dzekanaa/KevValueEngine/internal/wal"
)

const dirPermissions = 0o755 // rwxr-xr-x

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create data directories if they don't exist
	if err := os.MkdirAll(cfg.WALDir, dirPermissions); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create WAL directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.SSTableDir, dirPermissions); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create SSTable directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize BlockManager
	bm, err := blockmanager.New(cfg.BlockSize, cfg.CacheSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create Block Manager: %v\n", err)
		os.Exit(1)
	}

	// Initialize WAL
	walInstance := wal.NewWAL(cfg.WALDir, bm, cfg.BlockSize, cfg.RecordsPerSegment)

	// Recover WAL on startup
	walRecords, err := walInstance.Recover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to recover from WAL: %v\n", err)
	}

	// Initialize Memtable
	mt := memtable.New(cfg.MemtableMaxSize)

	// Initialize SSTable Manager
	sstMgr := sstable.NewManager(bm, cfg.SSTableDir, cfg.BlockSize)

	// Create Engine
	eng := engine.New(walInstance, mt, sstMgr, sstMgr)

	// Replay WAL records into Memtable during recovery
	if len(walRecords) > 0 {
		for _, record := range walRecords {
			if record.Tombstone {
				if err := eng.Delete(string(record.Key)); err != nil {
					fmt.Fprintf(os.Stderr, "recovery: failed to replay delete for key %s: %v\n", record.Key, err)
				}
			} else {
				if err := eng.Put(string(record.Key), record.Value); err != nil {
					fmt.Fprintf(os.Stderr, "recovery: failed to replay put for key %s: %v\n", record.Key, err)
				}
			}
		}
	}
}
