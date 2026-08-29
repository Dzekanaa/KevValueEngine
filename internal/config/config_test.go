package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileMissingReturnsDefaults(t *testing.T) {
	cfg, err := Load("does/not/exist.json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := defaultConfig()
	if *cfg != *want {
		t.Errorf("expected defaults %+v, got %+v", want, cfg)
	}
}

func TestLoadEmptyFileReturnsDefaults(t *testing.T) {
	path := writeConfig(t, `{}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := defaultConfig()
	if *cfg != *want {
		t.Errorf("expected defaults %+v, got %+v", want, cfg)
	}
}

func TestLoadPartialConfigOverridesOnlyGivenFields(t *testing.T) {
	path := writeConfig(t, `{"blockSize": 8192}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.BlockSize != 8192 {
		t.Errorf("expected BlockSize=8192, got %d", cfg.BlockSize)
	}

	want := defaultConfig()
	if cfg.RecordsPerSegment != want.RecordsPerSegment {
		t.Errorf("expected RecordsPerSegment to stay default (%d), got %d", want.RecordsPerSegment, cfg.RecordsPerSegment)
	}
	if cfg.MemtableMaxSize != want.MemtableMaxSize {
		t.Errorf("expected MemtableMaxSize to stay default (%d), got %d", want.MemtableMaxSize, cfg.MemtableMaxSize)
	}
}

func TestLoadFullConfigOverridesEverything(t *testing.T) {
	path := writeConfig(t, `{
		"blockSize": 8192,
		"recordsPerSegment": 500,
		"memtableMaxSize": 2000,
		"cacheSize": 200,
		"walDir": "custom/wal",
		"sstableDir": "custom/sstable"
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := Config{
		BlockSize:         8192,
		RecordsPerSegment: 500,
		MemtableMaxSize:   2000,
		CacheSize:         200,
		WALDir:            "custom/wal",
		SSTableDir:        "custom/sstable",
	}

	if *cfg != want {
		t.Errorf("expected %+v, got %+v", want, cfg)
	}
}

func TestLoadUnknownFieldReturnsError(t *testing.T) {
	path := writeConfig(t, `{"block_size": 8192}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestLoadMalformedJSONReturnsError(t *testing.T) {
	path := writeConfig(t, `{"blockSize": 8192,`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

// writeConfig writes the given content to a temp file and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}
