package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/retro-ware/wiz6-go/internal/game"
)

// TestUseSaveDir: the written files are copied into the save folder once, and an
// existing save is never overwritten.
func TestUseSaveDir(t *testing.T) {
	defer func() { game.SaveDir = "" }()
	data, save := t.TempDir(), filepath.Join(t.TempDir(), "save")
	os.WriteFile(filepath.Join(data, "pcfile.dbs"), []byte("roster"), 0o644) // DOS names in any case
	os.WriteFile(filepath.Join(data, "SCENARIO.HDR"), []byte("config"), 0o644)
	if err := useSaveDir(save, data); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(save, "PCFILE.DBS")); string(b) != "roster" {
		t.Fatalf("PCFILE.DBS = %q", b)
	}
	if _, err := os.Stat(filepath.Join(save, "SAVEGAME.DBS")); err == nil {
		t.Fatal("SAVEGAME.DBS made up")
	}
	os.WriteFile(filepath.Join(save, "SCENARIO.HDR"), []byte("changed"), 0o644)
	if err := useSaveDir(save, data); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(save, "SCENARIO.HDR")); string(b) != "changed" {
		t.Fatalf("SCENARIO.HDR overwritten: %q", b)
	}
	if game.SaveDir != save {
		t.Fatalf("SaveDir = %q", game.SaveDir)
	}
}
