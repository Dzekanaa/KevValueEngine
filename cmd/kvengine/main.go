package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

	// Start REPL (Read-Eval-Print Loop)
	fmt.Println("KeyValueEngine starting...")
	fmt.Println("Commands:\nPUT key value\nGET key\nDELETE key\nEXIT")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if err := handleCommand(eng, input); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scanner error: %v\n", err)
	}
}

// handleCommand parse and executes a single command.
func handleCommand(eng *engine.Engine, input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "PUT":
		if len(parts) < 3 {
			return fmt.Errorf("PUT requires: PUT key value")
		}
		key := parts[1]
		// Join remaining parts as value in case it contains spaces
		value := strings.Join(parts[2:], " ")

		if err := eng.Put(key, []byte(value)); err != nil {
			return fmt.Errorf("PUT failed: %w", err)
		}
		fmt.Printf("OK (key: %s)\n", key)

	case "GET":
		if len(parts) != 2 {
			return fmt.Errorf("GET requires: GET key")
		}
		key := parts[1]

		value, found := eng.Get(key)
		if !found {
			fmt.Printf("(nil)\n")
		} else {
			fmt.Printf("\"%s\"\n", string(value))
		}

	case "DELETE":
		if len(parts) != 2 {
			return fmt.Errorf("DELETE requires: DELETE key")
		}
		key := parts[1]

		if err := eng.Delete(key); err != nil {
			return fmt.Errorf("DELETE failed: %w", err)
		}
		fmt.Printf("OK (deleted: %s)\n", key)

	case "EXIT", "QUIT":
		fmt.Println("Exiting...")
		os.Exit(0)

	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}

	return nil
}
