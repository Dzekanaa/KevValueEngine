package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dzekanaa/KevValueEngine/internal/blockmanager"
	"github.com/Dzekanaa/KevValueEngine/internal/engine"
	"github.com/Dzekanaa/KevValueEngine/internal/memtable"
	"github.com/Dzekanaa/KevValueEngine/internal/sstable"
	"github.com/Dzekanaa/KevValueEngine/internal/wal"
)

// setupTestEngine creates a temporary engine for testing.
// It initializes all components (BlockManager, WAL, Memtable, SSTable)
// in a temporary directory that will be automatically cleaned up
// after the test completes.
func setupTestEngine(t *testing.T) (*engine.Engine, string) {
	t.Helper()

	tempDir := t.TempDir()
	walDir := filepath.Join(tempDir, "wal")
	sstDir := filepath.Join(tempDir, "sstable")

	if err := os.MkdirAll(walDir, dirPermissions); err != nil {
		t.Fatalf("failed to create WAL dir: %v", err)
	}
	if err := os.MkdirAll(sstDir, dirPermissions); err != nil {
		t.Fatalf("failed to create SSTable dir: %v", err)
	}

	bm, err := blockmanager.New(4096, 1000)
	if err != nil {
		t.Fatalf("failed to create block manager: %v", err)
	}

	walInstance := wal.NewWAL(walDir, bm, 4096, 1000)
	mt := memtable.New(100)
	sstMgr := sstable.NewManager(bm, sstDir, 4096)

	eng := engine.New(walInstance, mt, sstMgr, sstMgr)

	return eng, tempDir
}

// captureOutput captures stdout output from a function.
// Useful for testing console output without actually printing to terminal.
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestHandleCommand_PUT tests the PUT command with valid input.
// Verifies that a key-value pair is correctly stored and returns OK message.
func TestHandleCommand_PUT(t *testing.T) {
	eng, _ := setupTestEngine(t)

	output := captureOutput(func() {
		err := handleCommand(eng, "PUT user1 john_doe")
		if err != nil {
			t.Errorf("PUT failed: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("OK (key: user1)")) {
		t.Errorf("expected OK message, got: %s", output)
	}
}

// TestHandleCommand_GET tests the GET command with existing key.
// Verifies that the stored value is correctly retrieved.
func TestHandleCommand_GET(t *testing.T) {
	eng, _ := setupTestEngine(t)

	handleCommand(eng, "PUT testkey testvalue")

	output := captureOutput(func() {
		err := handleCommand(eng, "GET testkey")
		if err != nil {
			t.Errorf("GET failed: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("testvalue")) {
		t.Errorf("expected value 'testvalue', got: %s", output)
	}
}

// TestHandleCommand_GET_NotFound tests GET command with non-existent key.
// Verifies that (nil) is returned for keys that don't exist.
func TestHandleCommand_GET_NotFound(t *testing.T) {
	eng, _ := setupTestEngine(t)

	output := captureOutput(func() {
		err := handleCommand(eng, "GET nonexistent")
		if err != nil {
			t.Errorf("GET should not error: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("(nil)")) {
		t.Errorf("expected (nil), got: %s", output)
	}
}

// TestHandleCommand_DELETE tests the DELETE command.
// Verifies that a key is successfully marked for deletion.
func TestHandleCommand_DELETE(t *testing.T) {
	eng, _ := setupTestEngine(t)

	handleCommand(eng, "PUT delkey delvalue")

	output := captureOutput(func() {
		err := handleCommand(eng, "DELETE delkey")
		if err != nil {
			t.Errorf("DELETE failed: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("OK (deleted: delkey)")) {
		t.Errorf("expected OK message, got: %s", output)
	}

	// Verify key is actually deleted
	_, found := eng.Get("delkey")
	if found {
		t.Error("expected key to be deleted")
	}
}

// TestHandleCommand_DELETE_AfterDelete tests that GET returns nil after DELETE.
// Verifies the full lifecycle: PUT -> DELETE -> GET (nil).
func TestHandleCommand_DELETE_AfterDelete(t *testing.T) {
	eng, _ := setupTestEngine(t)

	handleCommand(eng, "PUT tempkey tempvalue")
	handleCommand(eng, "DELETE tempkey")

	output := captureOutput(func() {
		err := handleCommand(eng, "GET tempkey")
		if err != nil {
			t.Errorf("GET failed: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("(nil)")) {
		t.Errorf("expected (nil) after delete, got: %s", output)
	}
}

// TestHandleCommand_PUT_WithSpaces tests PUT command with values containing spaces.
// Verifies that multi-word values are correctly joined and stored.
func TestHandleCommand_PUT_WithSpaces(t *testing.T) {
	eng, _ := setupTestEngine(t)

	handleCommand(eng, "PUT greeting hello world from kvengine")

	value, found := eng.Get("greeting")
	if !found {
		t.Fatal("expected to find greeting key")
	}

	expected := "hello world from kvengine"
	if string(value) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(value))
	}
}

// TestHandleCommand_InvalidPUT tests PUT command with insufficient arguments.
// Verifies that error is returned when PUT lacks a value.
func TestHandleCommand_InvalidPUT(t *testing.T) {
	eng, _ := setupTestEngine(t)

	err := handleCommand(eng, "PUT onlykey")
	if err == nil {
		t.Fatal("expected error for invalid PUT")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("PUT requires")) {
		t.Errorf("expected PUT error message, got: %v", err)
	}
}

// TestHandleCommand_InvalidGET tests GET command with no arguments.
// Verifies that error is returned when GET lacks a key.
func TestHandleCommand_InvalidGET(t *testing.T) {
	eng, _ := setupTestEngine(t)

	err := handleCommand(eng, "GET")
	if err == nil {
		t.Fatal("expected error for invalid GET")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("GET requires")) {
		t.Errorf("expected GET error message, got: %v", err)
	}
}

// TestHandleCommand_InvalidDELETE tests DELETE command with no arguments.
// Verifies that error is returned when DELETE lacks a key.
func TestHandleCommand_InvalidDELETE(t *testing.T) {
	eng, _ := setupTestEngine(t)

	err := handleCommand(eng, "DELETE")
	if err == nil {
		t.Fatal("expected error for invalid DELETE")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("DELETE requires")) {
		t.Errorf("expected DELETE error message, got: %v", err)
	}
}

// TestHandleCommand_UnknownCommand tests handling of unknown commands.
// Verifies that error is returned for unsupported commands.
func TestHandleCommand_UnknownCommand(t *testing.T) {
	eng, _ := setupTestEngine(t)

	err := handleCommand(eng, "UNKNOWN")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("unknown command")) {
		t.Errorf("expected unknown command error, got: %v", err)
	}
}

// TestHandleCommand_FullScenario tests a complete workflow.
// Verifies that multiple PUT, GET, and DELETE operations work together correctly.
func TestHandleCommand_FullScenario(t *testing.T) {
	eng, _ := setupTestEngine(t)

	// PUT multiple keys
	cmds := []string{
		"PUT user:1 alice",
		"PUT user:2 bob",
		"PUT user:3 charlie",
	}

	for _, cmd := range cmds {
		if err := handleCommand(eng, cmd); err != nil {
			t.Errorf("command failed: %s, error: %v", cmd, err)
		}
	}

	// GET all keys and verify values
	testCases := []struct {
		key      string
		expected string
	}{
		{"user:1", "alice"},
		{"user:2", "bob"},
		{"user:3", "charlie"},
	}

	for _, tc := range testCases {
		value, found := eng.Get(tc.key)
		if !found {
			t.Errorf("expected to find key: %s", tc.key)
		}
		if string(value) != tc.expected {
			t.Errorf("expected '%s', got '%s'", tc.expected, string(value))
		}
	}

	// DELETE one key
	if err := handleCommand(eng, "DELETE user:3"); err != nil {
		t.Errorf("DELETE failed: %v", err)
	}

	// Verify deletion
	_, found := eng.Get("user:3")
	if found {
		t.Error("expected user:3 to be deleted")
	}
}

// TestHandleCommand_EmptyCommand tests empty command handling.
// Verifies that error is returned for empty input.
func TestHandleCommand_EmptyCommand(t *testing.T) {
	eng, _ := setupTestEngine(t)

	err := handleCommand(eng, "")
	if err == nil {
		t.Fatal("expected error for empty command")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("empty command")) {
		t.Errorf("expected empty command error, got: %v", err)
	}
}
