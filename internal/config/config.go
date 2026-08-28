package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// Config holds all user-configurable system parameters, with defaults
// applied for any field missing from the config file.
type Config struct {
	BlockSize         int    `json:"blockSize"`
	RecordsPerSegment int    `json:"recordsPerSegment"`
	MemtableMaxSize   int    `json:"memtableMaxSize"`
	CacheSize         int    `json:"cacheSize"`
	WALDir            string `json:"walDir"`
	SSTableDir        string `json:"sstableDir"`
}

// Load reads the config file at path and returns a Config with defaults
// applied for any field the file does not specify. If the file does not
// exist, it returns the defaults unchanged.
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// defaultConfig returns the built-in default values, used when the
// config file is missing or omits a field.
func defaultConfig() *Config {
	return &Config{
		BlockSize:         4096,
		RecordsPerSegment: 1000,
		MemtableMaxSize:   1000,
		CacheSize:         1000,
		WALDir:            "data/wal",
		SSTableDir:        "data/sstable",
	}
}
